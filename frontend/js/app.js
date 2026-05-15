const app = {
    async init() {
        state.subscribe(() => this.render());
        await state.init();
        this.bindEvents();
    },

    bindEvents() {
        // Auth Modal Close
        document.getElementById('overlay').onclick = () => {
            state.toggleAuthModal(false);
            state.toggleCart(false);
        };

        // Form Submission
        document.getElementById('login-form').onsubmit = async (e) => {
            e.preventDefault();
            const email = e.target.email.value;
            const password = e.target.password.value;
            try {
                await state.login(email, password);
                this.notify('Welcome back!', 'success');
            } catch (err) {
                this.notify(err.message, 'error');
            }
        };
    },

    async render() {
        const root = document.getElementById('app');
        const nav = document.getElementById('nav');
        const cartDrawer = document.getElementById('cart-drawer');
        const authModal = document.getElementById('auth-modal');
        const overlay = document.getElementById('overlay');

        // Nav
        nav.innerHTML = components.Navbar(state);

        // Views
        if (state.view === 'home') {
            root.innerHTML = components.Hero() + `
                <div class="container">
                    <div class="section-title">
                        <h2>Featured Drops</h2>
                        <a href="#" onclick="state.setView('catalog')" style="font-size: 0.9rem; color: var(--text-dim)">View All</a>
                    </div>
                    <div class="product-grid" id="featured-grid">Loading...</div>
                </div>
            `;
            this.loadProducts('featured-grid', 4);
        } else if (state.view === 'catalog') {
            root.innerHTML = `
                <div class="container">
                    <div class="section-title">
                        <h2>All Sneakers</h2>
                    </div>
                    <div class="product-grid" id="catalog-grid">Loading...</div>
                </div>
            `;
            this.loadProducts('catalog-grid');
        } else if (state.view === 'orders') {
            root.innerHTML = `
                <div class="container">
                    <div class="section-title">
                        <h2>Your Orders</h2>
                    </div>
                    <div id="orders-list">Loading...</div>
                </div>
            `;
            this.loadOrders();
        }

        // Cart
        if (state.isCartOpen) {
            cartDrawer.classList.add('active');
            overlay.classList.add('active');
            this.renderCart();
        } else {
            cartDrawer.classList.remove('active');
            if (!state.isAuthModalOpen) overlay.classList.remove('active');
        }

        // Auth Modal
        if (state.isAuthModalOpen) {
            authModal.classList.add('active');
            overlay.classList.add('active');
        } else {
            authModal.classList.remove('active');
            if (!state.isCartOpen) overlay.classList.remove('active');
        }
    },

    async loadProducts(elementId, limit = null) {
        const grid = document.getElementById(elementId);
        try {
            const raw = await api.products.list();
            let products = [];
            if (raw != null) {
                if (Array.isArray(raw)) products = raw;
                else if (raw.products && Array.isArray(raw.products)) products = raw.products;
            }
            if (limit) products = products.slice(0, limit);

            if (!grid) return;

            if (products.length === 0) {
                grid.innerHTML = '<div class="catalog-empty-state" role="status">No products yet.</div>';
                return;
            }

            grid.innerHTML = products.map(p => components.ProductCard({
                ...p,
                isFavorite: typeof state.isFavorite === 'function' && state.isFavorite(p.id)
            })).join('');
        } catch (err) {
            console.error(err);
            if (grid) grid.innerHTML = '<div class="catalog-empty-state" role="alert">Could not load products.</div>';
        }
    },

    async loadOrders() {
        try {
            const orders = await api.orders.list();
            const list = document.getElementById('orders-list');
            if (!list) return;

            if (orders.length === 0) {
                list.innerHTML = '<p style="color: var(--text-dim)">No orders found.</p>';
                return;
            }

            list.innerHTML = orders.map(o => `
                <div class="product-card" style="margin-bottom: 1rem; display: flex; justify-content: space-between; align-items: center; cursor: default">
                    <div>
                        <div style="font-weight: 800; margin-bottom: 0.5rem">Order #${o.id.substring(0, 8)}</div>
                        <div style="color: var(--text-dim); font-size: 0.8rem">${new Date(o.created_at).toLocaleDateString()}</div>
                    </div>
                    <div style="text-align: right">
                        <div class="product-price">$${o.total_amount.toFixed(2)}</div>
                        <div style="color: ${o.status === 'confirmed' ? '#2ecc71' : 'var(--accent)'}; text-transform: uppercase; font-size: 0.7rem; font-weight: 800; letter-spacing: 1px">${o.status}</div>
                    </div>
                </div>
            `).join('');
        } catch (err) {
            console.error(err);
        }
    },

    renderCart() {
        const list = document.getElementById('cart-list');
        const total = document.getElementById('cart-total');
        
        if (state.cart.length === 0) {
            list.innerHTML = '<p style="text-align: center; margin-top: 4rem; color: var(--text-dim)">Your cart is empty.</p>';
            total.innerHTML = '$0.00';
            document.getElementById('checkout-btn').disabled = true;
            return;
        }

        list.innerHTML = state.cart.map(item => components.CartItem(item)).join('');
        const sum = state.cart.reduce((acc, item) => acc + (item.unit_price * item.quantity), 0);
        total.innerHTML = `$${sum.toFixed(2)}`;
        document.getElementById('checkout-btn').disabled = false;
        document.getElementById('checkout-btn').onclick = () => this.checkout(sum);
    },

    async checkout(amount) {
        try {
            const order = await api.orders.checkout();
            this.notify('Order created! Processing payment...', 'success');
            
            await api.payments.pay({
                orderId: order.id,
                amount: amount,
                currency: 'USD',
                paymentMethod: 'credit_card'
            });

            this.notify('Payment successful! Order confirmed.', 'success');
            state.isCartOpen = false;
            await state.refreshCart();
            state.setView('orders');
        } catch (err) {
            this.notify(err.message, 'error');
        }
    },

    notify(message, type) {
        const toast = document.getElementById('notification');
        toast.innerHTML = message;
        toast.className = `notification active ${type}`;
        setTimeout(() => toast.classList.remove('active'), 3000);
    }
};

window.app = app;
app.init();
