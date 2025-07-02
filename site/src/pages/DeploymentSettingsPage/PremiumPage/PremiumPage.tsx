import { useDashboard } from "modules/dashboard/useDashboard";
import type { FC } from "react";
import { Helmet } from "react-helmet-async";
import { pageTitle } from "utils/page";
import { PremiumPageView } from "./PremiumPageView";

const PremiumPage: FC = () => {
	return (
		<>
			<Helmet>
				<title>{pageTitle("Features")}</title>
			</Helmet>
			<div style={{ padding: 24 }}>
				<h2 style={{ fontWeight: 600, fontSize: 22, margin: 0 }}>All features are available</h2>
				<p style={{ maxWidth: 460, fontSize: 14 }}>
					There are no feature restrictions or upgrade requirements. Enjoy full access.
				</p>
			</div>
		</>
	);
};

export default PremiumPage;
