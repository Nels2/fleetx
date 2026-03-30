package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// freescoutName is the name of the job as registered in the worker.
const freescoutName = "freescout"

var freeScoutTemplates = struct {
	VulnSummary              *template.Template
	VulnDescription          *template.Template
	FailingPolicySummary     *template.Template
	FailingPolicyDescription *template.Template
}{
	VulnSummary: template.Must(template.New("").Parse(
		`Vulnerability {{ .CVE }} detected on {{ len .Hosts }} host(s)`,
	)),

	// FreeScout supports markdown formatting.
	VulnDescription: template.Must(template.New("").Funcs(template.FuncMap{
		// CISAKnownExploit is *bool, so any condition check on it in the template
		// will test if nil or not, and not its actual boolean value. Hence, "deref".
		"deref": func(b *bool) bool { return *b },
	}).Parse(
		`See vulnerability (CVE) details in National Vulnerability Database (NVD) here: [{{ .CVE }}]({{ .NVDURL }}{{ .CVE }}).

{{ if .EPSSProbability }}
Probability of exploit (reported by [FIRST.org/epss](https://www.first.org/epss/)): {{ .EPSSProbability }}
{{ end }}
{{ if .CVSSScore }}CVSS score (reported by [NVD](https://nvd.nist.gov/)): {{ .CVSSScore }}
{{ end }}
{{ if .CVEPublished }}Published (reported by [NVD](https://nvd.nist.gov/)): {{ .CVEPublished }}
{{ end }}
{{ if .CISAKnownExploit }}Known exploits (reported by [CISA](https://www.cisa.gov/known-exploited-vulnerabilities-catalog)): {{ if deref .CISAKnownExploit }}Yes{{ else }}No{{ end }}
{{ end }}

Affected hosts:

{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .DisplayName }}]({{ $.FleetURL }}/hosts/{{ .ID }})
{{ range $path := .SoftwareInstalledPaths }}
    * {{ $path }}
{{ end }}
{{ end }}

View the affected software and more affected hosts:

1. Go to the [Software]({{ .FleetURL }}/software/manage) page in Fleet.
2. Above the list of software, in the **Search software** box, enter "{{ .CVE }}".
3. Hover over the affected software and select **View all hosts**.

----

This conversation was created automatically by your Fleet FreeScout integration.
`)),

	FailingPolicySummary: template.Must(template.New("").Parse(
		`{{ .PolicyName }} policy failed on {{ len .Hosts }} host(s)`,
	)),

	FailingPolicyDescription: template.Must(template.New("").Parse(
		`{{ if .PolicyCritical }}This policy is marked as **Critical** in Fleet.

{{ end }}Hosts:
{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .DisplayName }}]({{ $.FleetURL }}/hosts/{{ .ID }})
{{ end }}

View hosts that failed {{ .PolicyName }} on the [**Hosts**]({{ $.FleetURL }}/hosts/manage/?order_key=hostname&order_direction=asc&{{ if .TeamID }}team_id={{ .TeamID }}&{{ end }}policy_id={{ .PolicyID }}&policy_response=failing) page in Fleet.

----

This conversation was created automatically by your Fleet FreeScout integration.
`)),
}

type freeScoutVulnTplArgs struct {
	NVDURL   string
	FleetURL string
	CVE      string
	Hosts    []fleet.HostVulnerabilitySummary

	// Optional CVE metadata.
	EPSSProbability  *float64
	CVSSScore        *float64
	CISAKnownExploit *bool
	CVEPublished     *time.Time
}

// FreeScoutClient defines the method required for the client that makes API calls
// to FreeScout.
type FreeScoutClient interface {
	CreateFreeScoutConversation(ctx context.Context, subject, message string) (int64, error)
	FreeScoutConfigMatches(opts *externalsvc.FreeScoutOptions) bool
}

// FreeScout is the job processor for freescout integrations.
type FreeScout struct {
	FleetURL      string
	Datastore     fleet.Datastore
	Log           kitlog.Logger
	NewClientFunc func(*externalsvc.FreeScoutOptions) (FreeScoutClient, error)

	// mu protects concurrent access to clientsCache, so that the job processor
	// can potentially be run concurrently.
	mu sync.Mutex
	// map of integration type + team ID to FreeScout client (empty team ID for
	// global), e.g. "vuln:123", "failingPolicy:", etc.
	clientsCache map[string]FreeScoutClient
}

// Name returns the name of the job.
func (f *FreeScout) Name() string {
	return freescoutName
}

// returns nil, nil if there is no integration enabled for that message.
func (f *FreeScout) getClient(ctx context.Context, args freeScoutArgs) (FreeScoutClient, error) {
	var teamID uint
	var useTeamCfg bool

	intgType := args.integrationType()
	key := intgType + ":"
	if intgType == intgTypeFailingPolicy && args.FailingPolicy.TeamID != nil {
		teamID = *args.FailingPolicy.TeamID
		useTeamCfg = true
		key += fmt.Sprint(teamID)
	}

	ac, err := f.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	// load the config that would be used to create the client first - it is
	// needed to check if an existing client is configured the same or if its
	// configuration has changed since it was created.
	var opts *externalsvc.FreeScoutOptions
	if useTeamCfg {
		tm, err := f.Datastore.TeamLite(ctx, teamID)
		if err != nil {
			return nil, err
		}

		intgs, err := tm.Config.Integrations.MatchWithIntegrations(ac.Integrations)
		if err != nil {
			return nil, err
		}

		for _, intg := range intgs.Freescout {
			if intgType == intgTypeFailingPolicy && intg.EnableFailingPolicies {
				opts = &externalsvc.FreeScoutOptions{
					URL:           intg.URL,
					APIToken:      intg.APIToken,
					MailboxID:     intg.MailboxID,
					CustomerEmail: intg.CustomerEmail,
					AssignTo:      intg.AssignTo,
				}
				break
			}
		}
	} else {
		for _, intg := range ac.Integrations.Freescout {
			if (intgType == intgTypeVuln && intg.EnableSoftwareVulnerabilities) ||
				(intgType == intgTypeFailingPolicy && intg.EnableFailingPolicies) {
				opts = &externalsvc.FreeScoutOptions{
					URL:           intg.URL,
					APIToken:      intg.APIToken,
					MailboxID:     intg.MailboxID,
					CustomerEmail: intg.CustomerEmail,
					AssignTo:      intg.AssignTo,
				}
				break
			}
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.clientsCache == nil {
		f.clientsCache = make(map[string]FreeScoutClient)
	}
	if opts == nil {
		// no integration configured, clear any existing one
		delete(f.clientsCache, key)
		return nil, nil
	}

	// check if the existing one can be reused
	if cli := f.clientsCache[key]; cli != nil && cli.FreeScoutConfigMatches(opts) {
		return cli, nil
	}

	// otherwise create a new one
	cli, err := f.NewClientFunc(opts)
	if err != nil {
		return nil, err
	}
	f.clientsCache[key] = cli
	return cli, nil
}

// freeScoutArgs are the arguments for the FreeScout integration job.
type freeScoutArgs struct {
	Vulnerability *vulnArgs          `json:"vulnerability,omitempty"`
	FailingPolicy *failingPolicyArgs `json:"failing_policy,omitempty"`
}

func (a *freeScoutArgs) integrationType() string {
	if a.FailingPolicy == nil {
		return intgTypeVuln
	}
	return intgTypeFailingPolicy
}

// Run executes the freescout job.
func (f *FreeScout) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args freeScoutArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	cli, err := f.getClient(ctx, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get FreeScout client")
	}
	if cli == nil {
		// this message was queued when an integration was enabled, but since
		// then it has been disabled, so return success to mark the message
		// as processed.
		return nil
	}

	switch intgType := args.integrationType(); intgType {
	case intgTypeVuln:
		return f.runVuln(ctx, cli, args)
	case intgTypeFailingPolicy:
		return f.runFailingPolicy(ctx, cli, args)
	default:
		return ctxerr.Errorf(ctx, "unknown integration type: %v", intgType)
	}
}

func (f *FreeScout) runVuln(ctx context.Context, cli FreeScoutClient, args freeScoutArgs) error {
	vargs := args.Vulnerability
	if vargs == nil {
		return errors.New("invalid job args")
	}

	var hosts []fleet.HostVulnerabilitySummary
	var err error

	// Default to deprecated method in case we are processing an 'old' job payload
	// we are deprecating this because of performance reasons - querying by software_id should be
	// way more efficient than by CVE.
	if len(vargs.AffectedSoftwareIDs) == 0 {
		hosts, err = f.Datastore.HostsByCVE(ctx, vargs.CVE)
	} else {
		hosts, err = f.Datastore.HostVulnSummariesBySoftwareIDs(ctx, vargs.AffectedSoftwareIDs)
	}

	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching hosts")
	}

	tplArgs := &freeScoutVulnTplArgs{
		NVDURL:           nvdCVEURL,
		FleetURL:         f.FleetURL,
		CVE:              vargs.CVE,
		Hosts:            hosts,
		EPSSProbability:  vargs.EPSSProbability,
		CVSSScore:        vargs.CVSSScore,
		CISAKnownExploit: vargs.CISAKnownExploit,
		CVEPublished:     vargs.CVEPublished,
	}

	conversationID, err := f.createTemplatedConversation(ctx, cli, freeScoutTemplates.VulnSummary, freeScoutTemplates.VulnDescription, tplArgs)
	if err != nil {
		return err
	}
	level.Debug(f.Log).Log(
		"msg", "created freescout conversation for cve",
		"cve", vargs.CVE,
		"conversation_id", conversationID,
	)
	return nil
}

func (f *FreeScout) runFailingPolicy(ctx context.Context, cli FreeScoutClient, args freeScoutArgs) error {
	tplArgs := newFailingPoliciesTplArgs(f.FleetURL, args.FailingPolicy)

	conversationID, err := f.createTemplatedConversation(ctx, cli, freeScoutTemplates.FailingPolicySummary, freeScoutTemplates.FailingPolicyDescription, tplArgs)
	if err != nil {
		return err
	}

	attrs := []interface{}{
		"msg", "created freescout conversation for failing policy",
		"policy_id", args.FailingPolicy.PolicyID,
		"policy_name", args.FailingPolicy.PolicyName,
		"conversation_id", conversationID,
	}
	if args.FailingPolicy.TeamID != nil {
		attrs = append(attrs, "team_id", *args.FailingPolicy.TeamID)
	}
	level.Debug(f.Log).Log(attrs...)
	return nil
}

func (f *FreeScout) createTemplatedConversation(ctx context.Context, cli FreeScoutClient, summaryTpl, descTpl *template.Template, args interface{}) (int64, error) {
	var buf bytes.Buffer
	if err := summaryTpl.Execute(&buf, args); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "execute summary template")
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := descTpl.Execute(&buf, args); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "execute description template")
	}
	description := buf.String()

	conversationID, err := cli.CreateFreeScoutConversation(ctx, summary, description)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "create conversation")
	}
	return conversationID, nil
}

// QueueFreeScoutVulnJobs queues the FreeScout vulnerability jobs to process asynchronously
// via the worker.
func QueueFreeScoutVulnJobs(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	recentVulns []fleet.SoftwareVulnerability,
	cveMeta map[string]fleet.CVEMeta,
) error {
	level.Info(logger).Log("enabled", "true", "recentVulns", len(recentVulns))

	// for troubleshooting, log in debug level the CVEs that we will process
	// (cannot be done in the loop below as we want to add the debug log
	// _before_ we start processing them).
	cves := make([]string, 0, len(recentVulns))
	for _, vuln := range recentVulns {
		cves = append(cves, vuln.GetCVE())
	}
	sort.Strings(cves)
	level.Debug(logger).Log("recent_cves", fmt.Sprintf("%v", cves))

	cveGrouped := make(map[string][]uint)
	for _, v := range recentVulns {
		cveGrouped[v.GetCVE()] = append(cveGrouped[v.GetCVE()], v.Affected())
	}

	for cve, sIDs := range cveGrouped {
		args := vulnArgs{CVE: cve, AffectedSoftwareIDs: sIDs}
		if meta, ok := cveMeta[cve]; ok {
			args.EPSSProbability = meta.EPSSProbability
			args.CVSSScore = meta.CVSSScore
			args.CISAKnownExploit = meta.CISAKnownExploit
			args.CVEPublished = meta.Published
		}
		job, err := QueueJob(ctx, ds, freescoutName, freeScoutArgs{Vulnerability: &args})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}

// QueueFreeScoutFailingPolicyJob queues a FreeScout job for a failing policy to
// process asynchronously via the worker.
func QueueFreeScoutFailingPolicyJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
	policy *fleet.Policy, hosts []fleet.PolicySetHost,
) error {
	attrs := []interface{}{
		"enabled", "true",
		"failing_policy", policy.ID,
		"hosts_count", len(hosts),
	}
	if policy.TeamID != nil {
		attrs = append(attrs, "team_id", *policy.TeamID)
	}
	if len(hosts) == 0 {
		attrs = append(attrs, "msg", "skipping, no host")
		level.Debug(logger).Log(attrs...)
		return nil
	}

	level.Info(logger).Log(attrs...)

	args := &failingPolicyArgs{
		PolicyID:       policy.ID,
		PolicyName:     policy.Name,
		PolicyCritical: policy.Critical,
		TeamID:         policy.TeamID,
		Hosts:          hosts,
	}
	job, err := QueueJob(ctx, ds, freescoutName, freeScoutArgs{FailingPolicy: args})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
