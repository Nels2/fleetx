import React from "react";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ActionsDropdown from "components/ActionsDropdown";

import {
  IJiraIntegration,
  IZendeskIntegration,
  IFreeScoutIntegration,
  IIntegrationTableData as IIntegrationCompleteData,
} from "interfaces/integration";
import { IDropdownOption } from "interfaces/dropdownOption";

import JiraIcon from "../../../../../../assets/images/icon-jira-24x24@2x.png";
import ZendeskIcon from "../../../../../../assets/images/icon-zendesk-32x24@2x.png";
import FreeScoutIcon from "../../../../../../assets/images/icon-freescout-24x24.svg";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IIntegrationTableData;
  };
}
interface ICellProps extends IRowProps {
  cell: {
    value: string;
  };
}

interface IActionsDropdownProps extends IRowProps {
  cell: {
    value: IDropdownOption[];
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IActionsDropdownProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

export interface IIntegrationTableData extends IIntegrationCompleteData {
  actions: IDropdownOption[];
  name: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (
    value: string,
    integration: IIntegrationTableData
  ) => void
): IDataColumn[] => {
  const iconByType: Record<string, string> = {
    jira: JiraIcon,
    zendesk: ZendeskIcon,
    freescout: FreeScoutIcon,
  };
  const classByType: Record<string, string> = {
    jira: "jira-icon",
    zendesk: "zendesk-icon",
    freescout: "freescout-icon",
  };
  return [
    {
      title: "",
      Header: "",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "type",
      Cell: (cellProps: ICellProps) => {
        return (
          <div className="logo-cell">
            <img
              src={iconByType[cellProps.cell.value] || ZendeskIcon}
              alt="integration-icon"
              className={classByType[cellProps.cell.value] || "zendesk-icon"}
            />
          </div>
        );
      },
    },
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "name",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} className="w400" />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IActionsDropdownProps) => (
        <ActionsDropdown
          options={cellProps.cell.value}
          onChange={(value: string) =>
            actionSelectHandler(value, cellProps.row.original)
          }
          placeholder="Actions"
          variant="small-button"
        />
      ),
    },
  ];
};

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Delete",
      disabled: false,
      value: "delete",
    },
  ];
};

const enhanceJiraData = (
  jiraIntegrations: IJiraIntegration[]
): IIntegrationTableData[] => {
  return jiraIntegrations.map((integration, index) => {
    return {
      url: integration.url,
      username: integration.username,
      apiToken: integration.api_token,
      projectKey: integration.project_key,
      enableSoftwareVulnerabilities:
        integration.enable_software_vulnerabilities,
      name: `${integration.url} - ${integration.project_key}`,
      actions: generateActionDropdownOptions(),
      originalIndex: index,
      type: "jira",
    };
  });
};

const enhanceZendeskData = (
  zendeskIntegrations: IZendeskIntegration[]
): IIntegrationTableData[] => {
  return zendeskIntegrations.map((integration, index) => {
    return {
      url: integration.url,
      email: integration.email,
      apiToken: integration.api_token,
      groupId: integration.group_id,
      enableSoftwareVulnerabilities:
        integration.enable_software_vulnerabilities,
      name: `${integration.url} - ${integration.group_id}`,
      actions: generateActionDropdownOptions(),
      originalIndex: index,
      type: "zendesk",
    };
  });
};

const enhanceFreeScoutData = (
  freescoutIntegrations: IFreeScoutIntegration[]
): IIntegrationTableData[] => {
  return freescoutIntegrations.map((integration, index) => {
    return {
      url: integration.url,
      apiToken: integration.api_token,
      mailboxId: integration.mailbox_id,
      customerEmail: integration.customer_email,
      assignTo: integration.assign_to,
      enableSoftwareVulnerabilities:
        integration.enable_software_vulnerabilities,
      name: `${integration.url} - ${integration.mailbox_id}`,
      actions: generateActionDropdownOptions(),
      originalIndex: index,
      type: "freescout",
    };
  });
};

const combineDataSets = (
  jiraIntegrations: IJiraIntegration[],
  zendeskIntegrations: IZendeskIntegration[],
  freescoutIntegrations: IFreeScoutIntegration[]
): IIntegrationTableData[] => {
  const combine = [
    ...enhanceJiraData(jiraIntegrations),
    ...enhanceZendeskData(zendeskIntegrations),
    ...enhanceFreeScoutData(freescoutIntegrations),
  ];
  return combine.map((integration, index) => {
    return { ...integration, tableIndex: index };
  });
};

export { generateTableHeaders, combineDataSets };
