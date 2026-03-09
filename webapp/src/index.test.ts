interface WindowWithPlugin extends Window {
    registerPlugin: jest.Mock;
}

declare const window: WindowWithPlugin;

describe('Plugin', () => {
    beforeEach(() => {
        window.registerPlugin = jest.fn();
    });

    it('should register the plugin', async () => {
        await import('./index');
        expect(window.registerPlugin).toHaveBeenCalledWith(
            'co.appsome.claudecode',
            expect.any(Object)
        );
    });

    it('should have an initialize method', async () => {
        const module = await import('./index');
        const Plugin = module.default;
        const plugin = new Plugin();
        expect(typeof plugin.initialize).toBe('function');
    });
});
