// React component for in-app help, tooltips, and documentation for Agentic features

import React from "react";

interface AgenticHelpProps {
  topic?: string;
}

export const AgenticHelp: React.FC<AgenticHelpProps> = ({ topic }) => (
  <div style={{ background: "#f5f7fa", padding: "1em", borderRadius: "6px", margin: "1em 0" }}>
    <h4>Need Help?</h4>
    <p>
      {topic === "agent"
        ? "OpenCode agents enable advanced automation and orchestration in your workspace. Configure, monitor, and troubleshoot agents here."
        : topic === "workflow"
        ? "Agent-Zero workflows let you automate complex tasks. Monitor status, view logs, and manage workflow execution from this dashboard."
        : "Explore OpenCode and Agent-Zero features. Hover over info icons for tooltips, or visit the documentation for more details."}
    </p>
    <ul>
      <li>Getting started guides</li>
      <li>Feature documentation</li>
      <li>Troubleshooting tips</li>
    </ul>
    <a href="/docs/ai-coder/index" target="_blank" rel="noopener noreferrer">
      Full documentation
    </a>
  </div>
);
