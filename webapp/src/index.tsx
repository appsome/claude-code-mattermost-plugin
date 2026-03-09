// Plugin entry point
// This file will be loaded by Mattermost when the plugin is active

console.log('Claude Code plugin loaded');

// TODO: Add plugin initialization in Issue #5 (Interactive Components)
// - Register custom components
// - Set up state management
// - Initialize event listeners

export default class Plugin {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    public async initialize(_registry: unknown): Promise<void> {
        // Plugin initialization will be implemented in future issues
        console.log('Claude Code plugin initialized');
    }
}

// Export the plugin class for Mattermost to use
// eslint-disable-next-line @typescript-eslint/no-explicit-any
(window as unknown as {registerPlugin: (id: string, plugin: Plugin) => void}).registerPlugin('com.appsome.claudecode', new Plugin());
