/**
 * Svelte 5 Runes Admin State & Controller
 */

export class AdminState {
	isAdmin = $state(false);
	adminSecret = $state('');
	isModalOpen = $state(false);
	isCreating = $state(false);

	constructor() {
		if (typeof window !== 'undefined') {
			const saved = localStorage.getItem('lingua_admin_secret');
			if (saved) {
				this.adminSecret = saved;
				this.verify(saved);
			}
		}
	}

	async verify(secret: string): Promise<boolean> {
		try {
			const res = await fetch('/api/admin/verify', {
				headers: { 'X-Admin-Secret': secret }
			});
			if (res.ok) {
				this.isAdmin = true;
				this.adminSecret = secret;
				if (typeof window !== 'undefined') {
					localStorage.setItem('lingua_admin_secret', secret);
				}
				return true;
			}
		} catch (e) {
			console.error('Failed to verify admin', e);
		}
		this.isAdmin = false;
		return false;
	}

	logout() {
		this.isAdmin = false;
		this.adminSecret = '';
		if (typeof window !== 'undefined') {
			localStorage.removeItem('lingua_admin_secret');
		}
	}
}

export const adminState = new AdminState();

