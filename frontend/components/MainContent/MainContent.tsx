import classnames from "classnames";
import React, { ReactNode, useContext } from "react";

import AndroidEnterpriseDeletedMessage from "components/MDM/AndroidEnterpriseDeletedMessage";
import AppleBMRenewalMessage from "components/MDM/AppleBMRenewalMessage";
import AppleBMTermsMessage from "components/MDM/AppleBMTermsMessage";
import AppleBMTokenInvalidMessage from "components/MDM/AppleBMTokenInvalidMessage";
import ApplePNCertRenewalMessage from "components/MDM/ApplePNCertRenewalMessage";
import { AppContext } from "context/app";

import MicrosoftGraphCredentialInvalidMessage from "./banners/MicrosoftGraphCredentialInvalidMessage";
import VppRenewalMessage from "./banners/VppRenewalMessage";

export interface IMainContentConfig {
  renderedBanner: boolean;
}

interface IMainContentProps {
  children: ReactNode | ((mainContentConfig: IMainContentConfig) => ReactNode);
  /** Whether the page renders its own banner. Page banners take priority over
   * the license host limit banner, so it's hidden when this is true. */
  pageHasBanner?: boolean;
  /** An optional classname to pass to the main content component.
   * This can be used to apply styles directly onto the main content div
   */
  className?: string;
}

const baseClass = "main-content";

/**
 * A component that controls the layout and styling of the main content region
 * of the application.
 */
const MainContent = ({
  children,
  className,
}: IMainContentProps): JSX.Element => {
  const classes = classnames(baseClass, className);
  const {
    config,
    isAndroidEnterpriseDeleted,
    isApplePnsExpired,
    isAppleBmExpired,
    isVppExpired,
    needsAbmTermsRenewal,
    hasInvalidABMToken,
    invalidAbmTokenOrgNames,
    willAppleBmExpire,
    willApplePnsExpire,
    willVppExpire,
  } = useContext(AppContext);

  const renderAppWideBanner = () => {
    let banner: JSX.Element | null = null;

    // the order of these checks is important. This is the priority order
    // for showing banners and only one banner is shown at a time.
    if (isApplePnsExpired || willApplePnsExpire) {
      banner = <ApplePNCertRenewalMessage expired={isApplePnsExpired} />;
    } else if (isAndroidEnterpriseDeleted) {
      banner = <AndroidEnterpriseDeletedMessage />;
    } else if (isAppleBmExpired || willAppleBmExpire) {
      banner = <AppleBMRenewalMessage expired={isAppleBmExpired} />;
    } else if (needsAbmTermsRenewal) {
      banner = <AppleBMTermsMessage />;
    } else if (hasInvalidABMToken) {
      banner = (
        <AppleBMTokenInvalidMessage orgNames={invalidAbmTokenOrgNames} />
      );
    } else if (isVppExpired || willVppExpire) {
      banner = <VppRenewalMessage expired={isVppExpired} />;
    } else if (config?.mdm.microsoft_graph_credential_invalid) {
      banner = <MicrosoftGraphCredentialInvalidMessage />;
    }

    if (banner) {
      return (
        <div className={`${baseClass}--animation-disabled`}>
          <div className={`${baseClass}__warning-banner`}>{banner}</div>
        </div>
      );
    }

    return null;
  };

  const appWideBanner = renderAppWideBanner();
  const mainContentConfig: IMainContentConfig = {
    renderedBanner: !!appWideBanner,
  };

  return (
    <div className={classes}>
      {appWideBanner}
      {typeof children === "function" ? children(mainContentConfig) : children}
    </div>
  );
};

export default MainContent;
