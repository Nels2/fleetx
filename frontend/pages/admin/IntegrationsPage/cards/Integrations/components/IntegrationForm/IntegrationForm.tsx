import React, { useEffect, useState } from "react";

import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import { isValidEmail, isValidURL } from "components/forms/validators";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Spinner from "components/Spinner";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import {
  IIntegrationFormData,
  IIntegrationTableData,
  IIntegration,
  IZendeskJiraIntegrations,
  IIntegrationType,
} from "interfaces/integration";

const baseClass = "integration-form";

interface IIntegrationFormProps {
  onCancel: () => void;
  onSubmit: (
    integrationSubmitData: IIntegration[],
    integrationDestination: string
  ) => void | Promise<unknown>;
  integrationEditing?: IIntegrationTableData;
  integrations: IZendeskJiraIntegrations;
  integrationEditingUrl?: string;
  integrationEditingUsername?: string;
  integrationEditingEmail?: string;
  integrationEditingApiToken?: string;
  integrationEditingProjectKey?: string;
  integrationEditingGroupId?: number;
  integrationEditingMailboxId?: number;
  integrationEditingCustomerEmail?: string;
  integrationEditingAssignTo?: number;
  integrationEnableSoftwareVulnerabilities?: boolean;
  integrationEditingType?: IIntegrationType;
  destination?: string;
  testingConnection?: boolean;
  gitOpsModeEnabled?: boolean;
}

const validateForm = (
  data: IIntegrationFormData,
  destination: string
): IFormErrors => {
  const errors: IFormErrors = {};

  if (!data.url) {
    errors.url = "Enter a URL";
  } else if (!isValidURL({ url: data.url, protocols: ["https"] })) {
    errors.url = "Enter a valid HTTPS URL";
  }

  if (!data.apiToken) {
    errors.apiToken = "Enter an API token";
  }

  if (destination === "jira") {
    if (!data.username) {
      errors.username = "Enter a username";
    }
    if (!data.projectKey) {
      errors.projectKey = "Enter a project key";
    }
  } else if (destination === "freescout") {
    if (!data.customerEmail || !isValidEmail(data.customerEmail)) errors.customerEmail = "Enter a valid email";
    if (!data.mailboxId) errors.mailboxId = "Enter a mailbox ID";
  } else {
    if (!data.email) {
      errors.email = "Enter an email";
    } else if (!isValidEmail(data.email)) {
      errors.email = "Enter a valid email";
    }
    if (!data.groupId) {
      errors.groupId = "Enter a group ID";
    }
  }

  return errors;
};

const IntegrationForm = ({
  onCancel,
  onSubmit,
  integrationEditing,
  integrations,
  integrationEditingUrl,
  integrationEditingUsername,
  integrationEditingEmail,
  integrationEditingApiToken,
  integrationEditingProjectKey,
  integrationEditingGroupId,
  integrationEditingMailboxId,
  integrationEditingCustomerEmail,
  integrationEditingAssignTo,
  integrationEnableSoftwareVulnerabilities,
  integrationEditingType,
  destination,
  testingConnection,
  gitOpsModeEnabled,
}: IIntegrationFormProps): JSX.Element => {
  const { jira: jiraIntegrations, zendesk: zendeskIntegrations, freescout: freescoutIntegrations } = integrations;

  const [integrationDestination, setIntegrationDestination] = useState(
    integrationEditingType || destination || "jira"
  );

  useEffect(() => {
    setIntegrationDestination(destination || integrationEditingType || "jira");
  }, [destination, integrationEditingType]);

  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    getTextFieldProps,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IIntegrationFormData>({
    initialFormData: {
      url: integrationEditingUrl || "",
      username: integrationEditingUsername || "",
      email: integrationEditingEmail || "",
      apiToken: integrationEditingApiToken || "",
      projectKey: integrationEditingProjectKey || "",
      groupId: integrationEditingGroupId || 0,
      mailboxId: integrationEditingMailboxId || 0,
      customerEmail: integrationEditingCustomerEmail || "",
      assignTo: integrationEditingAssignTo || 0,
      enableSoftwareVulnerabilities:
        integrationEnableSoftwareVulnerabilities || false,
    },
    validate: (data) => validateForm(data, integrationDestination),
    isSubmitting: testingConnection,
    skipTrim: ["apiToken"],
  });

  // Flipping destination unmounts the other shape's fields; drop their errors
  // so a lingering value can't reappear on a flip back.
  useEffect(() => {
    if (integrationDestination === "jira") {
      clearFieldError("email");
      clearFieldError("groupId");
    } else {
      clearFieldError("username");
      clearFieldError("projectKey");
    }
  }, [integrationDestination, clearFieldError]);

  // IntegrationForm component can be used to create a new integration or edit
  // an existing integration, so submitData will be assembled accordingly.
  const createSubmitData = (data: IIntegrationFormData): IIntegration[] => {
    let jiraIntegrationSubmitData = jiraIntegrations || [];
    let zendeskIntegrationSubmitData = zendeskIntegrations || [];
    let freescoutIntegrationSubmitData = freescoutIntegrations || [];

    // Editing through UI is temporarily deprecated in 4.14
    if (integrationDestination === "jira") {
      if (
        integrationEditing &&
        (integrationEditing.originalIndex ||
          integrationEditing.originalIndex === 0) &&
        integrationEditing.username
      ) {
        // Edit existing jira integration using array replacement
        jiraIntegrationSubmitData.splice(integrationEditing.originalIndex, 1, {
          url: data.url,
          username: data.username || "",
          api_token: data.apiToken,
          project_key: data.projectKey || "",
        });
      } else {
        // Create new jira integration at end of array
        jiraIntegrationSubmitData = [
          ...jiraIntegrationSubmitData,
          {
            url: data.url,
            username: data.username || "",
            api_token: data.apiToken,
            project_key: data.projectKey || "",
          },
        ];
      }
      return jiraIntegrationSubmitData;
    }
    if (integrationDestination === "freescout") {
      if (
        integrationEditing &&
        (integrationEditing.originalIndex ||
          integrationEditing.originalIndex === 0) &&
        integrationEditing.customerEmail
      ) {
        // Edit existing freescout integration using array replacement
        freescoutIntegrationSubmitData.splice(
          integrationEditing.originalIndex,
          1,
          {
            url: data.url,
            api_token: data.apiToken,
            mailbox_id: parseInt(data.mailboxId as any, 10) || 0,
            customer_email: data.customerEmail || "",
            assign_to: parseInt(data.assignTo as any, 10) || 0,
          }
        );
      } else {
        // Create new freescout integration at end of array
        freescoutIntegrationSubmitData = [
          ...freescoutIntegrationSubmitData,
          {
            url: data.url,
            api_token: data.apiToken,
            mailbox_id: parseInt(data.mailboxId as any, 10) || 0,
            customer_email: data.customerEmail || "",
            assign_to: parseInt(data.assignTo as any, 10) || 0,
          },
        ];
      }
      return freescoutIntegrationSubmitData;
    }
    if (
      integrationEditing &&
      (integrationEditing.originalIndex ||
        integrationEditing.originalIndex === 0) &&
      integrationEditing.email
    ) {
      // Edit existing zendesk integration using array replacement
      zendeskIntegrationSubmitData.splice(integrationEditing.originalIndex, 1, {
        url: data.url,
        email: data.email || "",
        api_token: data.apiToken,
        group_id: Number(data.groupId) || 0,
      });
    } else {
      // Create new zendesk integration at end of array
      zendeskIntegrationSubmitData = [
        ...zendeskIntegrationSubmitData,
        {
          url: data.url,
          email: data.email || "",
          api_token: data.apiToken,
          group_id: Number(data.groupId) || 0,
        },
      ];
    }
    return zendeskIntegrationSubmitData;
  };

  const onValidSubmit = (data: IIntegrationFormData) =>
    onSubmit(createSubmitData(data), integrationDestination);

  if (testingConnection) {
    return (
      <div className={`${baseClass}__testing-connection`}>
        <strong>Testing connection</strong>
        <Spinner />
      </div>
    );
  }

  return (
    <form
      className={`${baseClass}__form`}
      onSubmit={handleSubmit(onValidSubmit)}
      autoComplete="off"
      noValidate
    >
      <InputField
        autofocus
        {...getTextFieldProps("url")}
        label="URL"
        placeholder={
          integrationDestination === "jira"
            ? "https://example.atlassian.net"
            : "https://example.zendesk.com"
        }
        disabled={gitOpsModeEnabled || isSubmitting}
      />
      {integrationDestination === "jira" ? (
        <InputField
          {...getTextFieldProps("username")}
          label="Username"
          placeholder="name@example.com"
          disabled={gitOpsModeEnabled || isSubmitting}
        />
      ) : integrationDestination === "freescout" ? (
        <InputField {...getTextFieldProps("customerEmail")} label="Customer email" type="email" disabled={gitOpsModeEnabled || isSubmitting} />
      ) : (
        <InputField
          {...getTextFieldProps("email")}
          label="Email"
          placeholder="name@example.com"
          type="email"
          disabled={gitOpsModeEnabled || isSubmitting}
        />
      )}
      <InputField
        {...getTextFieldProps("apiToken")}
        label="API token"
        disabled={gitOpsModeEnabled || isSubmitting}
      />
      {integrationDestination === "jira" ? (
        <InputField
          {...getTextFieldProps("projectKey")}
          label="Project key"
          placeholder="JRAEXAMPLE"
          disabled={gitOpsModeEnabled || isSubmitting}
          tooltip={
            <>
              To find the Jira project key, head to your project in Jira. Your
              project key is located in the URL. For example, in
              &ldquo;jira.example.com/projects/JRAEXAMPLE,&rdquo;
              &ldquo;JRAEXAMPLE&rdquo; is your project key.
            </>
          }
        />
      ) : integrationDestination === "freescout" ? (
        <>
          <InputField name="mailboxId" label="Mailbox ID" type="number" value={formData.mailboxId || ""} onChange={(value: string) => setField("mailboxId", Number(value) || 0)} onFocus={() => clearFieldError("mailboxId")} onBlur={() => validateField("mailboxId")} error={getError("mailboxId")} disabled={gitOpsModeEnabled || isSubmitting} />
          <InputField name="assignTo" label="Assign to (optional)" type="number" value={formData.assignTo || ""} onChange={(value: string) => setField("assignTo", Number(value) || 0)} disabled={gitOpsModeEnabled || isSubmitting} />
        </>
      ) : (
        <InputField
          name="groupId"
          label="Group ID"
          placeholder="28134038"
          type="number"
          value={formData.groupId || ""}
          onChange={(value: string) =>
            setField("groupId", value ? Number(value) : 0)
          }
          onFocus={() => clearFieldError("groupId")}
          onBlur={() => validateField("groupId")}
          error={getError("groupId")}
          disabled={gitOpsModeEnabled || isSubmitting}
          tooltip={
            <>
              To find the Zendesk group ID, select{" "}
              <strong>Admin &gt; People &gt; Groups</strong>. Find the group and
              select it. The group ID will appear in the search field.
            </>
          }
        />
      )}
      <div className="modal-cta-wrap">
        <GitOpsModeTooltipWrapper
          tipOffset={8}
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={disableChildren || isSubmitting}
              isLoading={isSubmitting}
            >
              Add
            </Button>
          )}
        />
        <Button onClick={onCancel} variant="secondary">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default IntegrationForm;
