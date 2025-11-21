import { render } from 'ink';
import React from 'react';
import App from './App';

export function launchTUI(): void {
  try {
    const { unmount, waitUntilExit } = render(React.createElement(App));

    // Keep the process running until Ink unmounts (which happens on q/Esc)
    waitUntilExit().catch((error) => {
      console.error('TUI error:', error);
      process.exit(1);
    });
  } catch (error) {
    console.error('Failed to launch TUI:', error);
    process.exit(1);
  }
}
