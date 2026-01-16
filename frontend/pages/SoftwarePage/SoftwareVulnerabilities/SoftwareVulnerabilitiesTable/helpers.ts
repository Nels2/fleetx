export const getExploitedVulnerabilitiesDropdownOptions = () => {
  return [
    {
      isDisabled: false,
      label: "All vulnerabilities",
      value: "false",
      helpText: "All vulnerabilities detected on your hosts.",
    },
    {
      label: "Exploited vulnerabilities",
      value: "true",
      helpText:
        "Vulnerabilities that have been actively exploited in the wild.",
    },
  ];
};

export const isValidCVEFormat = (query: string): boolean => {
  if (query.length < 9) {
    return false;
  }

  const cveRegex = /^CVE-\d{4}-\d{4,}$/i;
  return cveRegex.test(query);
};
