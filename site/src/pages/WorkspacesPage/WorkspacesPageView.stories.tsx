import type { Meta, StoryObj } from "@storybook/react";
import { expect, within } from "@storybook/test";
import {
	type Workspace,
	type WorkspaceStatus,
	WorkspaceStatuses,
} from "api/typesGenerated";
import {
	MockMenu,
	getDefaultFilterProps,
} from "components/Filter/storyHelpers";
import { DEFAULT_RECORDS_PER_PAGE } from "components/PaginationWidget/utils";
import dayjs from "dayjs";
import uniqueId from "lodash/uniqueId";
import type { ComponentProps } from "react";
import {
	MockBuildInfo,
	MockOrganization,
	MockPendingProvisionerJob,
	MockStoppedWorkspace,
	MockTemplate,
	MockUserOwner,
	MockWorkspace,
	MockWorkspaceAgent,
	MockWorkspaceAppStatus,
	mockApiError,
} from "testHelpers/entities";
import {
	withAuthProvider,
	withDashboardProvider,
	withProxyProvider,
} from "testHelpers/storybook";
import { WorkspacesPageView } from "./WorkspacesPageView";

const createWorkspace = (
	name: string,
	status: WorkspaceStatus,
	outdated = false,
	lastUsedAt = "0001-01-01",
	dormantAt?: string,
	deletingAt?: string,
): Workspace => {
	return {
		...MockWorkspace,
		id: uniqueId("workspace"),
		name: name,
		outdated,
		latest_build: {
			...MockWorkspace.latest_build,
			status,
			job:
				status === "pending"
					? MockPendingProvisionerJob
					: MockWorkspace.latest_build.job,
		},
		last_used_at: lastUsedAt,
		dormant_at: dormantAt || null,
		deleting_at: deletingAt || null,
	};
};

// This is type restricted to prevent future statuses from slipping
// through the cracks unchecked!
const workspaces = WorkspaceStatuses.map((status) =>
	createWorkspace(status, status),
);

// Additional Workspaces depending on time
const additionalWorkspaces: Record<string, Workspace> = {
	today: createWorkspace(
		"running-outdated",
		"running",
		true,
		dayjs().subtract(3, "hour").toString(),
	),
	old: createWorkspace(
		"old-outdated",
		"running",
		true,
		dayjs().subtract(1, "week").toString(),
	),
	oldStopped: createWorkspace(
		"old-stopped-outdated",
		"stopped",
		true,
		dayjs().subtract(1, "week").toString(),
	),
	oldRequireActiveVersion: {
		...createWorkspace(
			"old-require-active-version-outdated",
			"running",
			true,
			dayjs().subtract(1, "week").toString(),
		),
		template_require_active_version: true,
	},
	oldStoppedRequireActiveVersion: {
		...createWorkspace(
			"old-stopped-require-active-version-outdated",
			"stopped",
			true,
			dayjs().subtract(1, "week").toString(),
		),
		template_require_active_version: true,
	},
	veryOld: createWorkspace(
		"very-old-running-outdated",
		"running",
		true,
		dayjs().subtract(1, "month").subtract(4, "day").toString(),
	),
};

const dormantWorkspaces: Record<string, Workspace> = {
	dormantNoDelete: createWorkspace(
		"dormant-no-delete",
		"stopped",
		false,
		dayjs().subtract(1, "month").toString(),
		dayjs().subtract(1, "month").toString(),
	),
	dormantAutoDelete: createWorkspace(
		"dormant-auto-delete",
		"stopped",
		false,
		dayjs().subtract(1, "month").toString(),
		dayjs().subtract(1, "month").toString(),
		dayjs().add(29, "day").toString(),
	),
};

const allWorkspaces = [
	...Object.values(workspaces),
	...Object.values(additionalWorkspaces),
];

type FilterProps = ComponentProps<typeof WorkspacesPageView>["filterProps"];

const defaultFilterProps = getDefaultFilterProps<FilterProps>({
	query: "owner:me",
	menus: {
		user: MockMenu,
		template: MockMenu,
		status: MockMenu,
		organizations: MockMenu,
	},
	values: {
		owner: MockUserOwner.username,
		template: undefined,
		status: undefined,
	},
});

const mockTemplates = [
	MockTemplate,
	...[1, 2, 3, 4].map((num) => {
		return {
			...MockTemplate,
			active_user_count: Math.floor(Math.random() * 10) * num,
			display_name: `Extra Template ${num}`,
			description: "Auto-Generated template",
			icon: num % 2 === 0 ? "" : "/icon/goland.svg",
		};
	}),
];

const meta: Meta<typeof WorkspacesPageView> = {
	title: "pages/WorkspacesPage",
	component: WorkspacesPageView,
	args: {
		limit: DEFAULT_RECORDS_PER_PAGE,
		filterProps: defaultFilterProps,
		checkedWorkspaces: [],
		canCheckWorkspaces: true,
		templates: mockTemplates,
		templatesFetchStatus: "success",
		count: 13,
		page: 1,
	},
	parameters: {
		queries: [
			{
				key: ["buildInfo"],
				data: MockBuildInfo,
			},
		],
		user: MockUserOwner,
	},
	decorators: [withAuthProvider, withDashboardProvider, withProxyProvider()],
};

export default meta;
type Story = StoryObj<typeof WorkspacesPageView>;

export const AllStates: Story = {
	args: {
		workspaces: allWorkspaces,
		count: allWorkspaces.length,
	},
};

export const AllStatesWithFavorites: Story = {
	args: {
		workspaces: allWorkspaces.map((workspace, i) => ({
			...workspace,
			// NOTE: testing sort order is not relevant here.
			favorite: i % 2 === 0,
		})),
		count: allWorkspaces.length,
	},
};

const icons = [
	"/icon/code.svg",
	"/icon/aws.svg",
	"/icon/docker-white.svg",
	"/icon/docker.svg",
	"",
	"/icon/doesntexist.svg",
];

export const Icons: Story = {
	args: {
		workspaces: allWorkspaces.map((workspace, i) => ({
			...workspace,
			template_icon: icons[i % icons.length],
		})),
		count: allWorkspaces.length,
	},
};

export const OwnerHasNoWorkspaces: Story = {
	args: {
		workspaces: [],
		count: 0,
		canCreateTemplate: true,
	},
};

export const OwnerHasNoWorkspacesAndNoTemplates: Story = {
	args: {
		workspaces: [],
		templates: [],
		count: 0,
		canCreateTemplate: true,
	},
};

export const UserHasNoWorkspaces: Story = {
	args: {
		workspaces: [],
		count: 0,
		canCreateTemplate: false,
	},
};

export const UserHasNoWorkspacesAndNoTemplates: Story = {
	args: {
		workspaces: [],
		templates: [],
		count: 0,
		canCreateTemplate: false,
	},
};

export const NoSearchResults: Story = {
	args: {
		workspaces: [],
		filterProps: {
			...defaultFilterProps,
			filter: {
				...defaultFilterProps.filter,
				query: "searchwithnoresults",
				used: true,
			},
		},
		count: 0,
	},
};

export const UnhealthyWorkspace: Story = {
	args: {
		workspaces: [
			{
				...createWorkspace("unhealthy", "running"),
				health: {
					healthy: false,
					failing_agents: [],
				},
			},
		],
	},
};

export const DormantWorkspaces: Story = {
	args: {
		workspaces: Object.values(dormantWorkspaces),
		count: Object.values(dormantWorkspaces).length,
	},
};

export const WithError: Story = {
	args: {
		error: mockApiError({ message: "Something went wrong" }),
	},
};

export const InvalidPageNumber: Story = {
	args: {
		workspaces: [],
		count: 200,
		limit: 25,
		page: 1000,
	},
};

export const MultipleApps: Story = {
	args: {
		workspaces: [
			{
				...MockWorkspace,
				name: "multiple-apps",
				latest_build: {
					...MockWorkspace.latest_build,
					resources: [
						{
							...MockWorkspace.latest_build.resources[0],
							agents: [
								{
									...MockWorkspaceAgent,
									apps: [
										{
											...MockWorkspaceAgent.apps[0],
											display_name: "App 1",
											id: "app-1",
										},
										{
											...MockWorkspaceAgent.apps[0],
											display_name: "App 2",
											id: "app-2",
										},
									],
								},
							],
						},
					],
				},
			},
		],
		count: allWorkspaces.length,
	},
};

export const ShowOrganizations: Story = {
	args: {
		workspaces: [
			{
				...MockWorkspace,
				name: "other-org-workspace",
				organization_name: "limbus-co",
			},
		],
	},

	parameters: {
		showOrganizations: true,
		organizations: [
			{
				...MockOrganization,
				name: "limbus-co",
				display_name: "Limbus Company, LLC",
			},
		],
	},

	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const accessibleTableCell = await canvas.findByRole("cell", {
			// The organization label is always visually hidden, but the test
			// makes sure that there's a screen reader hook to give the table
			// cell more structured output
			name: /organization: Limbus Company, LLC/i,
		});

		expect(accessibleTableCell).toBeDefined();
	},
};

export const WithLatestAppStatus: Story = {
	args: {
		workspaces: [
			{
				...MockWorkspace,
				name: "long-app-status",
				latest_app_status: {
					...MockWorkspaceAppStatus,
					message:
						"This is a long message that will wrap around the component. It should wrap many times because this is very very very very very long.",
				},
			},
			{
				...MockWorkspace,
				name: "no-app-status",
				latest_app_status: null,
			},
			{
				...MockWorkspace,
				name: "app-status-working",
				latest_app_status: {
					...MockWorkspaceAppStatus,
					state: "working",
					message: "Fixing the competitors page...",
				},
			},
			{
				...MockWorkspace,
				name: "app-status-failure",
				latest_app_status: {
					...MockWorkspaceAppStatus,
					state: "failure",
					message: "I couldn't figure it out...",
				},
			},
			{
				...{
					...MockStoppedWorkspace,
					latest_build: {
						...MockStoppedWorkspace.latest_build,
						resources: [],
					},
				},
				name: "stopped-app-status-failure",
				latest_app_status: {
					...MockWorkspaceAppStatus,
					state: "failure",
					message: "I couldn't figure it out...",
					uri: "",
				},
			},
			{
				...MockWorkspace,
				name: "app-status-working-with-uri",
				latest_app_status: {
					...MockWorkspaceAppStatus,
					state: "working",
					message: "Updating the README...",
					uri: "file:///home/coder/projects/coder/coder/README.md",
				},
			},
		],
	},
};
