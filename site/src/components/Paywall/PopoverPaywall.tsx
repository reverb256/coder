import type { Interpolation, Theme } from "@emotion/react";
import Link from "@mui/material/Link";
import { PremiumBadge } from "components/Badges/Badges";
import { Button } from "components/Button/Button";
import { Stack } from "components/Stack/Stack";
import { CircleCheckBigIcon } from "lucide-react";
import type { FC, ReactNode } from "react";

interface PopoverPaywallProps {
	message: string;
	description?: ReactNode;
	documentationLink?: string;
}

export const PopoverPaywall: FC<PopoverPaywallProps> = ({
	message,
	description,
	documentationLink,
}) => {
	return (
		<div style={{ padding: 24 }}>
			<h5 style={{ fontWeight: 600, fontSize: 18, margin: 0 }}>{message}</h5>
			{description && <p style={{ maxWidth: 360, fontSize: 14 }}>{description}</p>}
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
	root: {
		display: "flex",
		flexDirection: "row",
		alignItems: "center",
		maxWidth: 770,
		padding: "24px 36px",
		borderRadius: 8,
		gap: 18,
	},
	title: {
		fontWeight: 600,
		fontFamily: "inherit",
		fontSize: 18,
		margin: 0,
	},
	description: (theme) => ({
		marginTop: 8,
		fontFamily: "inherit",
		maxWidth: 360,
		lineHeight: "160%",
		color: theme.palette.text.secondary,
		fontSize: 14,
	}),
	separator: (theme) => ({
		width: 1,
		height: 180,
		backgroundColor: theme.palette.divider,
		marginLeft: 8,
	}),
	featureList: {
		listStyle: "none",
		margin: 0,
		marginRight: 8,
		padding: "0 0 0 24px",
		fontSize: 13,
		fontWeight: 500,
	},
	learnButton: {
		padding: "0 28px",
	},
	feature: {
		display: "flex",
		alignItems: "center",
		padding: 3,
		gap: 8,
		lineHeight: 1.2,
	},
} satisfies Record<string, Interpolation<Theme>>;
