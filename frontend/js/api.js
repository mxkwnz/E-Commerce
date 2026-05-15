const API_BASE = window.location.origin;

const api = {
    async request(path, options = {}) {
        const token = localStorage.getItem('token');
        const headers = {
            'Content-Type': 'application/json',
            ...(token && token !== 'undefined' && token !== 'null' ? { 'Authorization': `Bearer ${token}` } : {}),
            ...options.headers
        };

        let response;
        try {
            response = await fetch(`${API_BASE}${path}`, { ...options, headers });
        } catch (networkErr) {
            throw new Error('Network error: ' + networkErr.message);
        }

        // Handle empty responses (e.g. 204 No Content)
        const text = await response.text();
        let data = null;
        if (text) {
            try { data = JSON.parse(text); } catch { data = { error: text }; }
        }

        if (!response.ok) {
            throw new Error((data && data.error) || `HTTP ${response.status}`);
        }

        return data;
    },

    auth: {
        login: (email, password) => api.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password })
        }),
        register: (userData) => api.request('/auth/register', {
            method: 'POST',
            body: JSON.stringify(userData)
        }),
        validate: () => api.request('/auth/validate', { method: 'POST' }),
        forgotPassword: (email) => api.request('/auth/forgot-password', {
            method: 'POST',
            body: JSON.stringify({ email })
        }),
        resetPassword: (token, newPassword) => api.request('/auth/reset-password', {
            method: 'POST',
            body: JSON.stringify({ token, newPassword })
        }),
        changePasswordSendCode: () => api.request('/auth/change-password/send-code', { method: 'POST' }),
        changePasswordConfirm: (code, newPassword) => api.request('/auth/change-password/confirm', {
            method: 'POST',
            body: JSON.stringify({ code, newPassword })
        })
    },

    products: {
        list: (params = {}) => {
            const q = new URLSearchParams();
            Object.entries(params).forEach(([k, v]) => {
                if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
            });
            const qs = q.toString();
            return api.request('/products' + (qs ? '?' + qs : ''));
        },
        get: (id) => api.request(`/products/${id}`),
        add: (productData) => api.request('/products', { method: 'POST', body: JSON.stringify(productData) }),
        update: (id, productData) => api.request(`/products/${id}`, { method: 'PUT', body: JSON.stringify(productData) }),
        delete: (id) => api.request(`/products/${id}`, { method: 'DELETE' })
    },

    cart: {
        get: () => api.request('/cart-items'),
        add: (productId, quantity = 1, price = 0, size = null) => api.request('/cart-items', {
            method: 'POST',
            body: JSON.stringify({ product_id: productId, quantity, unitPrice: price, size })
        }),
        update: (id, quantity) => api.request(`/cart-items/${id}`, {
            method: 'PUT',
            body: JSON.stringify({ quantity })
        }),
        remove: (id) => api.request(`/cart-items/${id}`, { method: 'DELETE' })
    },

    orders: {
        checkout: () => api.request('/orders/checkout', { method: 'POST' }),
        list: () => api.request('/orders'),
        get: (id) => api.request(`/orders/${id}`)
    },

    payments: {
        pay: (paymentData) => api.request('/payments/pay', {
            method: 'POST',
            body: JSON.stringify(paymentData)
        })
    },

    users: {
        list: () => api.request('/users'),
        get: (id) => api.request(`/users/${id}`),
        create: (userData) => api.request('/users', { method: 'POST', body: JSON.stringify(userData) }),
        update: (id, userData) => api.request(`/users/${id}`, {
            method: 'PUT',
            body: JSON.stringify(userData)
        }),
        topUpBalance: (userId, amount) => api.request(`/users/${userId}/balance/top-up`, {
            method: 'POST',
            body: JSON.stringify({ amount })
        })
    },

    favorites: {
        list: () => api.request('/favorites'),
        add: (productId) => api.request(`/products/${productId}/favorite`, { method: 'POST' }),
        remove: (productId) => api.request(`/products/${productId}/favorite`, { method: 'DELETE' })
    },

    reviews: {
        add: (productId, rating, comment) => api.request('/reviews', {
            method: 'POST',
            body: JSON.stringify({ productId, rating, comment })
        }),
        update: (id, rating, comment) => api.request(`/reviews/${id}`, {
            method: 'PUT',
            body: JSON.stringify({ rating, comment })
        }),
        delete: (id) => api.request(`/reviews/${id}`, { method: 'DELETE' }),
        list: (params = {}) => {
            const q = new URLSearchParams();
            Object.entries(params).forEach(([k, v]) => {
                if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
            });
            const qs = q.toString();
            return api.request('/reviews' + (qs ? '?' + qs : ''));
        },
        get: (id) => api.request(`/reviews/${id}`),
        listByProduct: (productId) => api.request(`/products/${productId}/reviews`),
        listByUser: (userId) => api.request(`/users/${userId}/reviews`)
    }
};

window.api = api;
