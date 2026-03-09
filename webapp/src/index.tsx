// Plugin entry point
// This file will be loaded by Mattermost when the plugin is active

console.log('Claude Code plugin loaded');

// TODO: Add plugin initialization in Issue #5 (Interactive Components)
// - Register custom components
// - Set up state management
// - Initialize event listeners

export default class Plugin {
    public async initialize(registry: any) {
        // Plugin initialization will be implemented in future issues
        console.log('Claude Code plugin initialized');
    }
}

// Export the plugin class for Mattermost to use
(window as any).registerPlugin('com.appsome.claudecode', new Plugin());
