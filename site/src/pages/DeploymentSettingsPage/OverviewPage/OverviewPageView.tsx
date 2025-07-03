import AlertTitle from "@mui/material/AlertTitle";
import type {
	DAUsResponse,
	Experiment,
	SerpentOption,
} from "api/typesGenerated";
import { Link } from "components/Link/Link";
import {
	SettingsHeader,
	SettingsHeaderDescription,
	SettingsHeaderDocsLink,
	SettingsHeaderTitle,
} from "components/SettingsHeader/SettingsHeader";
import { Stack } from "components/Stack/Stack";
import type { FC } from "react";
import { useDeploymentOptions } from "utils/deployOptions";
import { docs } from "utils/docs";
import { Alert } from "../../../components/Alert/Alert";
import OptionsTable from "../OptionsTable";
import { UserEngagementChart } from "./UserEngagementChart";
<<<<<<< HEAD
import { AgentList } from "../../../components/Agentic/AgentList";
import { AgentWorkflowMonitor } from "../../../components/Agentic/AgentWorkflowMonitor";
import { AgenticHelp } from "../../../components/Agentic/AgenticHelp";
=======
>>>>>>> upstream/main

type OverviewPageViewProps = {
	deploymentOptions: SerpentOption[];
	dailyActiveUsers: DAUsResponse | undefined;
	readonly invalidExperiments: readonly string[];
	readonly safeExperiments: readonly Experiment[];
};

export const OverviewPageView: FC<OverviewPageViewProps> = ({
	deploymentOptions,
	dailyActiveUsers,
	safeExperiments,
	invalidExperiments,
}) => {
	return (
		<>
			<SettingsHeader
				actions={<SettingsHeaderDocsLink href={docs("/admin/setup")} />}
			>
				<SettingsHeaderTitle>General</SettingsHeaderTitle>
				<SettingsHeaderDescription>
					Information about your Coder deployment.
				</SettingsHeaderDescription>
			</SettingsHeader>

			<Stack spacing={4}>
				<UserEngagementChart
					data={dailyActiveUsers?.entries.map((i) => ({
						date: i.date,
						users: i.amount,
					}))}
				/>
<<<<<<< HEAD
				<AgentList
					agents={[
						{ id: "1", name: "OpenCode Alpha", status: "online", type: "OpenCode" },
						{ id: "2", name: "Agent-Zero", status: "offline", type: "Agent-Zero" },
					]}
				/>
				<AgentWorkflowMonitor
					tasks={[
						{ id: "wf1", name: "Build Project", status: "completed", startedAt: "2025-07-02T12:00:00Z", finishedAt: "2025-07-02T12:05:00Z" },
						{ id: "wf2", name: "Run Tests", status: "running", startedAt: "2025-07-02T12:10:00Z" },
					]}
				/>
				<AgenticHelp topic="agent" />
=======
>>>>>>> upstream/main
				{invalidExperiments.length > 0 && (
					<Alert severity="warning">
						<AlertTitle>Invalid experiments in use:</AlertTitle>
						<ul>
							{invalidExperiments.map((it) => (
								<li key={it}>
									<pre>{it}</pre>
								</li>
							))}
						</ul>
						It is recommended that you remove these experiments from your
						configuration as they have no effect. See{" "}
						<Link
							href="https://coder.com/docs/cli/server#--experiments"
							target="_blank"
							rel="noreferrer"
						>
							the documentation
						</Link>{" "}
						for more details.
					</Alert>
				)}
				<OptionsTable
					options={useDeploymentOptions(
						deploymentOptions,
						"Access URL",
						"Wildcard Access URL",
						"Experiments",
					)}
					additionalValues={safeExperiments}
				/>
			</Stack>
		</>
	);
};
