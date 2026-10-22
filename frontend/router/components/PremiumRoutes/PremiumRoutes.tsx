import React, { useContext } from "react";

import { AppContext } from "context/app";

interface IAuthenticatedAdminRoutesProps {
  children: JSX.Element;
}

/**
 * Retains the signed-in requirement for legacy premium routes
 */
const PremiumRoutes = ({ children }: IAuthenticatedAdminRoutesProps) => {
  const { currentUser } = useContext(AppContext);

  if (!currentUser) {
    return null;
  }

  return <>{children}</>;
};

export default PremiumRoutes;
