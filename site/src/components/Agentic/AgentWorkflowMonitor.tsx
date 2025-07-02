// React component to monitor Agent-Zero workflows and orchestration tasks

import React from "react";

export interface WorkflowTask {
  id: string;
  name: string;
  status: "pending" | "running" | "completed" | "failed";
  startedAt: string;
  finishedAt?: string;
}

interface AgentWorkflowMonitorProps {
  tasks: WorkflowTask[];
  onSelectTask?: (task: WorkflowTask) => void;
}

export const AgentWorkflowMonitor: React.FC<AgentWorkflowMonitorProps> = ({ tasks, onSelectTask }) => (
  <div>
    <h3>Agent-Zero Workflows</h3>
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>Status</th>
          <th>Started</th>
          <th>Finished</th>
        </tr>
      </thead>
      <tbody>
        {tasks.map(task => (
          <tr key={task.id} onClick={() => onSelectTask?.(task)} style={{ cursor: onSelectTask ? "pointer" : "default" }}>
            <td>{task.name}</td>
            <td>{task.status}</td>
            <td>{task.startedAt}</td>
            <td>{task.finishedAt || "-"}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);
