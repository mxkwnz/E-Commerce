/**
 * Global state manager. Handles auth session + cart across all pages.
 * Uses localStorage for token persistence. Never calls logout() unless
 * the server explicitly returns a non-200 on /auth/validate.
 */
const state = {
    user: null,
    cart: [],
    favoriteProductIds: new Set(),
    listeners: [],

    subscribe(fn) {
        this.listeners.push(fn);
    },

    notify() {
        this.listeners.forEach(fn => fn(this));
    },

    /**
     * Called once on DOMContentLoaded. Reads token from localStorage,
     * validates it against the server, loads cart if valid.
     * Only calls _clearSession() if the token is provably invalid (401).
     */
    async init() {
        const token = localStorage.getItem('token');
        if (!token || token === 'undefined' || token === 'null') {
            // No token stored at all - user is a guest
            this.user = null;
            this.cart = [];
            this.favoriteProductIds = new Set();
            this.notify();
            return;
        }

        try {
            const user = await api.auth.validate();
            if (user && user.id) {
                this.user = user;
                await this._loadCart();
                await this._loadFavorites();
            } else {
                // validate returned 200 but no user - should never happen
                this._clearSession();
            }
        } catch (err) {
            // Only clear session for auth errors (401), not network errors
            const msg = err.message || '';
            if (msg.includes('401') || msg.includes('invalid') || msg.includes('expired') || msg.includes('token')) {
                console.warn('[state] Token invalid, clearing session:', msg);
                this._clearSession();
            } else {
                // Network error or 5xx - keep the user state as-is, don't log out
                console.warn('[state] Validate failed (network?), keeping session:', msg);
                // Still try to restore user from a previous known-good state
                const savedUserId = localStorage.getItem('userId');
                if (savedUserId) {
                    this.user = { id: savedUserId };
                }
            }
        }
        this.notify();
    },

    _clearSession() {
        localStorage.removeItem('token');
        localStorage.removeItem('userId');
        this.user = null;
        this.cart = [];
        this.favoriteProductIds = new Set();
    },

    isFavorite(productId) {
        return this.favoriteProductIds && this.favoriteProductIds.has(productId);
    },

    async _loadFavorites() {
        this.favoriteProductIds = new Set();
        if (!this.user) return;
        try {
            const data = await api.favorites.list();
            const ids = (data && data.productIds) ? data.productIds : [];
            this.favoriteProductIds = new Set(ids);
        } catch (err) {
            console.warn('[state] Favorites load failed:', err.message);
        }
    },

    async toggleFavorite(productId) {
        if (!this.user) {
            showToast('Please login to save favorites.', 'error');
            setTimeout(() => { window.location.href = 'auth.html'; }, 1200);
            return;
        }
        try {
            if (this.favoriteProductIds.has(productId)) {
                await api.favorites.remove(productId);
                this.favoriteProductIds.delete(productId);
                showToast('Removed from favorites', 'success');
            } else {
                await api.favorites.add(productId);
                this.favoriteProductIds.add(productId);
                showToast('Saved to favorites', 'success');
            }
            this.notify();
            if (typeof window.refreshAccountFavorites === 'function') {
                window.refreshAccountFavorites();
            }
        } catch (err) {
            showToast('Favorites: ' + (err.message || 'failed'), 'error');
        }
    },

    async _loadCart() {
        try {
            const items = await api.cart.get();
            this.cart = Array.isArray(items) ? items : [];
        } catch (err) {
            console.warn('[state] Cart load failed:', err.message);
            this.cart = [];
        }
    },

    async login(email, password) {
        const resp = await api.auth.login(email, password);
        const token = resp.accessToken;
        const userId = resp.userId;
        if (!token) throw new Error('No token received from server');
        localStorage.setItem('token', token);
        localStorage.setItem('userId', userId || '');
        const user = await api.auth.validate();
        this.user = user;
        await this._loadCart();
        await this._loadFavorites();
        this.notify();
    },

    async register(userData) {
        const resp = await api.auth.register(userData);
        const token = resp.accessToken;
        const userId = resp.userId;
        if (!token) throw new Error('No token received from server');
        localStorage.setItem('token', token);
        localStorage.setItem('userId', userId || '');
        const user = await api.auth.validate();
        this.user = user;
        this.cart = [];
        await this._loadFavorites();
        this.notify();
    },

    logout() {
        this._clearSession();
        this.notify();
    },

    async addToCart(productId, price = 0, size = null) {
        if (!this.user) {
            showToast('Please login to add items to your cart.', 'error');
            setTimeout(() => window.location.href = 'auth.html', 1500);
            return;
        }
        try {
            await api.cart.add(productId, 1, price, size);
            await this._loadCart();
            this.notify();
            showToast('Added to cart!', 'success');
            // Reset global size selection
            window._selectedSize = null;
        } catch (err) {
            showToast('Failed to add to cart: ' + err.message, 'error');
        }
    },

    async removeFromCart(itemId) {
        try {
            await api.cart.remove(itemId);
            await this._loadCart();
            this.notify();
        } catch (err) {
            showToast('Failed to remove item: ' + err.message, 'error');
        }
    },

    async updateCartQuantity(itemId, quantity) {
        if (quantity < 1) {
            return this.removeFromCart(itemId);
        }
        try {
            await api.cart.update(itemId, quantity);
            await this._loadCart();
            this.notify();
        } catch (err) {
            showToast('Failed to update quantity: ' + err.message, 'error');
        }
    },
};

function showToast(msg, type = 'success') {
    const toast = document.getElementById('notification');
    if (!toast) return;
    toast.innerHTML = msg;
    toast.className = `notification active ${type}`;
    clearTimeout(toast._timer);
    toast._timer = setTimeout(() => toast.classList.remove('active'), 3500);
}

window.state = state;
window.showToast = showToast;

/** Favorite heart on product cards — must use data-product-id (inline JSON broke onclick HTML). */
function handleProductFavoriteClick(ev) {
    ev.preventDefault();
    ev.stopPropagation();
    const el = ev.currentTarget;
    const raw = el && el.getAttribute('data-product-id');
    if (!raw || !window.state) return;
    const id = decodeURIComponent(raw);
    window.state.toggleFavorite(id);
}
window.handleProductFavoriteClick = handleProductFavoriteClick;
