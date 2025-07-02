// React component for configuring agent settings and preferences

import React from "react";

export interface AgentConfig {
  id: string;
  name: string;
  enabled: boolean;
  config: Record<string, any>;
}

interface AgentConfigPanelProps {
  agent: AgentConfig;
  onUpdateConfig: (config: AgentConfig) => void;
}

export const AgentConfigPanel: React.FC<AgentConfigPanelProps> = ({ agent, onUpdateConfig }) => {
  const [enabled, setEnabled] = React.useState(agent.enabled);

  const handleToggle = () => {
    setEnabled(!enabled);
    onUpdateConfig({ ...agent, enabled: !enabled });
  };

  return (
    <div>
      <h3>Agent Settings: {agent.name}</h3>
      <label>
        <input
          type="checkbox"
          checked={enabled}
          onChange={handleToggle}
        />
        Enabled
      </label>
      {/* Render agent-specific config fields here */}
      <pre>{JSON.stringify(agent.config, null, 2)}</pre>
    </div>
  );
};
