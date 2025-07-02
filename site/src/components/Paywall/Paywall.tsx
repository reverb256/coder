import type { Interpolation, Theme } from "@emotion/react";
import Link from "@mui/material/Link";
import { PremiumBadge } from "components/Badges/Badges";
import { Button } from "components/Button/Button";
import { Stack } from "components/Stack/Stack";
import { CircleCheckBigIcon } from "lucide-react";
import type { FC, ReactNode } from "react";

interface PaywallProps {
	message: string;
	description?: ReactNode;
	documentationLink?: string;
}

export const Paywall: FC<PaywallProps> = ({
	message,
	description,
	documentationLink,
}) => {
	return (
		<div style={{ padding: 24 }}>
			<h5 style={{ fontWeight: 600, fontSize: 22, margin: 0 }}>{message}</h5>
			{description && <p style={{ maxWidth: 460, fontSize: 14 }}>{description}</p>}
			{documentationLink && (
				<a
					href={documentationLink}
					target="_blank"
					rel="noreferrer"
					style={{ fontWeight: 600 }}
				>
					Read the documentation
				</a>
			)}
		</div>
	);
};

const FeatureIcon: FC = () => {
	return (
		<CircleCheckBigIcon
			aria-hidden="true"
			className="size-icon-sm"
			css={[
				(theme) => ({
					color: theme.branding.premium.border,
				}),
			]}
		/>
	);
};

const styles = {
	root: (theme) => ({
		display: "flex",
		flexDirection: "row",
		justifyContent: "center",
		alignItems: "center",
		minHeight: 280,
		padding: 24,
		borderRadius: 8,
		gap: 32,
		backgroundImage: `linear-gradient(160deg, transparent, ${theme.branding.premium.background})`,
		border: `1px solid ${theme.branding.premium.border}`,
	}),
	title: {
		fontWeight: 600,
		fontFamily: "inherit",
		fontSize: 22,
		margin: 0,
	},
	description: () => ({
		fontFamily: "inherit",
		maxWidth: 460,
		fontSize: 14,
	}),
	separator: (theme) => ({
		width: 1,
		height: 220,
		backgroundColor: theme.branding.premium.divider,
		marginLeft: 8,
	}),
	learnButton: {
		padding: "0 28px",
	},
	featureList: {
		listStyle: "none",
		margin: 0,
		padding: "0 24px",
		fontSize: 14,
		fontWeight: 500,
	},
	featureIcon: (theme) => ({
		color: theme.roles.active.fill.outline,
	}),
	feature: {
		display: "flex",
		alignItems: "center",
		padding: 3,
		gap: 8,
	},
} satisfies Record<string, Interpolation<Theme>>;
