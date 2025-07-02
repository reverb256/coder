// React component for viewing agent performance metrics and logs

import React from "react";

export interface AgentLogEntry {
  timestamp: string;
  level: "info" | "warning" | "error";
  message: string;
}

interface AgentLogsProps {
  logs: AgentLogEntry[];
}

export const AgentLogs: React.FC<AgentLogsProps> = ({ logs }) => (
  <div>
    <h3>Agent Logs</h3>
    <table>
      <thead>
        <tr>
          <th>Time</th>
          <th>Level</th>
          <th>Message</th>
        </tr>
      </thead>
      <tbody>
        {logs.map((log, idx) => (
          <tr key={idx}>
            <td>{log.timestamp}</td>
            <td>{log.level}</td>
            <td>{log.message}</td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);
