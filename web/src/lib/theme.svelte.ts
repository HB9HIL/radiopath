const root = document.documentElement;

/** The light/dark theme: set before first paint in app.html, switched by the user, otherwise following the system. */
class Theme {
	dark = $state(root.dataset.theme === 'dark');

	/** Follows changes of the system theme while the user has not picked one. */
	constructor() {
		matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
			try {
				if (localStorage.getItem('cp-theme')) return;
			} catch {
				return;
			}
			this.set(e.matches);
		});
	}

	/** Applies a theme to the page without remembering it. */
	set(dark: boolean) {
		this.dark = dark;
		root.dataset.theme = dark ? 'dark' : 'light';
	}

	/** Switches between light and dark and remembers the choice. */
	toggle() {
		this.set(!this.dark);
		try {
			localStorage.setItem('cp-theme', this.dark ? 'dark' : 'light');
		} catch {}
	}
}

export const theme = new Theme();
