// React component to list OpenCode agents and their status

import React from "react";
import { StatusIndicator } from "../StatusIndicator/StatusIndicator";

export interface Agent {
  id: string;
  name: string;
  status: "online" | "offline" | "error";
  type: string;
}

interface AgentListProps {
  agents: Agent[];
  onSelectAgent?: (agent: Agent) => void;
}

export const AgentList: React.FC<AgentListProps> = ({ agents, onSelectAgent }) => (
  <div>
    <h3>OpenCode Agents</h3>
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>Type</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        {agents.map(agent => (
          <tr key={agent.id} onClick={() => onSelectAgent?.(agent)} style={{ cursor: onSelectAgent ? "pointer" : "default" }}>
            <td>{agent.name}</td>
            <td>{agent.type}</td>
            <td>
              <StatusIndicator status={agent.status} />
              {agent.status}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);
