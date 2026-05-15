const components = {
    ProductCard(p) {
        const img = p.photoUrl || p.image_url || 'premium_sneaker_1.png';
        const price = typeof p.price === 'number' ? p.price.toFixed(2) : parseFloat(p.price || 0).toFixed(2);

        let starsHtml = '';
        if (p.reviews && p.reviews.length > 0) {
            const avg = p.reviews.reduce((s, r) => s + (r.rating || 0), 0) / p.reviews.length;
            const full = Math.round(avg);
            starsHtml = `<div class="product-reviews">
                <span style="color:#f39c12;">${'★'.repeat(full)}${'☆'.repeat(5-full)}</span>
                <span style="margin-left:0.35rem;color:var(--text-dim);font-size:0.75rem;">(${p.reviews.length})</span>
            </div>`;
        }

        const favClass = p.isFavorite ? ' fav-active' : '';
        const pidEnc = encodeURIComponent(String(p.id));

        return `
            <div class="product-card" onclick="if(window.openProductDetail) window.openProductDetail('${p.id}')">
                <div class="product-img-wrapper">
                    <img src="${img}" alt="${p.name}" onerror="this.src='premium_sneaker_1.png'">
                </div>
                <div class="product-info">
                    ${p.brand ? `<div class="brand-tag">${p.brand}</div>` : ''}
                    <h3>${p.name}</h3>
                    <div class="product-price">$${price}</div>
                    ${starsHtml}
                </div>
                <div class="product-card-footer">
                    <button type="button" class="add-btn" onclick="event.stopPropagation(); state.addToCart('${p.id}', ${p.price || 0})" title="Add to cart">+</button>
                    <button type="button" class="fav-btn${favClass} product-fav-btn" data-product-id="${pidEnc}" title="Favorite" onclick="event.stopPropagation(); handleProductFavoriteClick(event)">♥</button>
                    <button type="button" class="review-btn" onclick="event.stopPropagation(); if(window.openReviewModal) window.openReviewModal('${p.id}')" title="Write review">★</button>
                </div>
            </div>
        `;
    },

    ProductDetailModal(p) {
        const img = p.photoUrl || p.image_url || 'premium_sneaker_1.png';
        const price = typeof p.price === 'number' ? p.price.toFixed(2) : parseFloat(p.price || 0).toFixed(2);
        const favClass = p.isFavorite ? ' fav-active' : '';

        const reviewsHtml = (p.reviews && p.reviews.length > 0) 
            ? p.reviews.map(r => {
                const isOwner = state.user && state.user.id === r.userId;
                const isAdmin = state.user && state.user.role === 'admin';
                const showDelete = isOwner || isAdmin;
                
                return `
                <div style="padding: 1.5rem 0; border-bottom: 1px solid var(--border); position: relative;">
                    <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                        <div>
                            <div style="color:#f39c12; margin-bottom: 0.5rem; font-size: 0.9rem;">${'★'.repeat(r.rating)}${'☆'.repeat(5-r.rating)}</div>
                            <p style="font-size: 1rem; color: var(--text); line-height: 1.5;">${r.comment}</p>
                            <div style="font-size: 0.75rem; color: var(--text-dim); margin-top: 0.5rem;">
                                ${new Date(r.createdAt).toLocaleDateString()} ${isOwner ? '<span style="color: var(--accent); margin-left: 0.5rem; font-weight: 600;">(Your Review)</span>' : ''}
                            </div>
                        </div>
                        ${showDelete ? `
                            <button class="action-btn btn-delete" 
                                    style="padding: 0.4rem 0.8rem; font-size: 0.75rem; border: 1px solid rgba(231, 76, 60, 0.2); border-radius: 6px;" 
                                    onclick="window.deleteReview('${r.id}', '${p.id}')">Delete</button>
                        ` : ''}
                    </div>
                </div>
                `;
            }).join('')
            : '<p style="color: var(--text-dim); padding: 1rem 0;">No reviews yet. Be the first to review!</p>';

        return `
            <div class="product-visual">
                <img src="${img}" alt="${p.name}" style="max-width: 100%; filter: drop-shadow(0 20px 40px rgba(0,0,0,0.5));">
            </div>
            <div class="product-details">
                <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1.5rem;">
                    <div>
                        <div class="brand-tag">${p.brand || 'Premium'}</div>
                        <h2>${p.name}</h2>
                        <div class="product-price" style="font-size: 2rem;">$${price}</div>
                    </div>
                    <button class="btn-outline" style="padding: 0.5rem; border-radius: 50%;" onclick="closeProductDetail()">&times;</button>
                </div>
                
                <p style="color: var(--text-dim); margin-bottom: 2rem; font-size: 1rem; line-height: 1.6;">${p.description || 'Elevate your style with these premium sneakers. Crafted for comfort and designed for the streets, they feature top-tier materials and a classic silhouette that never goes out of style.'}</p>
                
                <div style="margin-bottom: 2.5rem;">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                        <span style="font-weight: 700; text-transform: uppercase; font-size: 0.8rem; letter-spacing: 1px;">Select Size (EU)</span>
                        <span style="color: var(--text-dim); font-size: 0.8rem; cursor: pointer; text-decoration: underline;">Size Guide</span>
                    </div>
                    <div class="size-grid" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(60px, 1fr)); gap: 0.75rem;">
                        ${(p.sizes && p.sizes.length > 0 
                            ? p.sizes 
                            : (p.gender === 'women' ? ['36', '37', '38', '39', '40', '41'] : ['40', '41', '42', '43', '44', '45'])
                          ).map(s => `
                            <div class="size-option" onclick="document.querySelectorAll('.size-option').forEach(el=>el.classList.remove('selected')); this.classList.add('selected'); window._selectedSize='${s}'" 
                                 style="padding: 0.85rem; border: 1px solid var(--border); border-radius: 8px; text-align: center; cursor: pointer; font-weight: 600; transition: var(--transition);">
                                ${s}
                            </div>
                        `).join('')}
                    </div>
                </div>
                
                <div style="display: flex; gap: 1rem; margin-bottom: 3rem;">
                    <button class="btn btn-primary" style="flex: 2; border-radius: 12px;" onclick="if(!window._selectedSize){ showToast('Please select a size', 'error'); return; } state.addToCart('${p.id}', ${p.price || 0}, window._selectedSize)">Add to Cart</button>
                    <button class="fav-btn${favClass}" style="flex: 0 0 54px; height: 54px; border-radius: 12px; font-size: 1.5rem;" data-product-id="${p.id}" onclick="handleProductFavoriteClick(event)">♥</button>
                </div>
                
                <div style="margin-top: 2rem;">
                    <h3 style="text-transform: uppercase; letter-spacing: 2px; font-size: 0.9rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; margin-bottom: 1rem;">Customer Reviews</h3>
                    ${reviewsHtml}
                </div>
            </div>
        `;
    },

    CartItem(item, product) {
        const price = typeof item.unitPrice === 'number' ? item.unitPrice.toFixed(2) : '0.00';
        const p = product || {};
        const img = p.photoUrl || p.image_url || 'premium_sneaker_1.png';
        const title = p.name || `Item #${item.productId ? item.productId.substring(0, 8) : ''}...`;
        const brandLine = p.brand ? `<p style="margin:0.25rem 0 0;font-size:0.85rem;color:var(--text-dim);">${p.brand}</p>` : '';
        return `
            <div class="cart-item">
                <img src="${img}" alt="${title.replace(/"/g, '&quot;')}" onerror="this.src='premium_sneaker_1.png'">
                <div class="cart-item-info">
                    <div class="cart-item-header">
                        <div>
                            <h4 style="margin:0;">${title}</h4>
                            ${brandLine}
                            <p style="margin:0.25rem 0 0;font-size:0.8rem;color:var(--accent);font-weight:600;">Size: ${item.size || 'N/A'}</p>
                        </div>
                        <span class="product-price">$${price}</span>
                    </div>
                    <div class="cart-item-controls">
                        <div class="quantity-selector">
                            <button class="qty-btn" onclick="state.updateCartQuantity('${item.id}', ${item.quantity - 1})">-</button>
                            <span class="qty-val">${item.quantity}</span>
                            <button class="qty-btn" onclick="state.updateCartQuantity('${item.id}', ${item.quantity + 1})">+</button>
                        </div>
                        <button class="remove-btn" onclick="state.removeFromCart('${item.id}')">Remove</button>
                    </div>
                </div>
            </div>
        `;
    },

    Navbar(state) {
        const cartCount = (state.cart && state.cart.length) || 0;
        const userSection = state.user
            ? `<a href="account.html" class="icon-btn" style="text-decoration:none;">
                   &#128100; ${state.user.username || 'Profile'}
               </a>
               <button class="btn btn-outline" style="padding:0.4rem 1rem;font-size:0.85rem;" onclick="state.logout();window.location.href='index.html'">Logout</button>`
            : `<a href="auth.html" class="btn btn-outline" style="padding:0.5rem 1.5rem;text-decoration:none;">Login</a>`;
        const adminLink = state.user && state.user.role === 'admin' ? `<li><a href="admin.html" style="color: var(--accent); font-weight: bold;">Admin</a></li>` : '';
        return `
            <div class="logo" onclick="window.location.href='index.html'" style="cursor:pointer">SNEAKR</div>
            <ul class="nav-links">
                <li><a href="index.html">Home</a></li>
                <li><a href="catalog.html">Catalog</a></li>
                ${adminLink}
            </ul>
            <div class="nav-icons">
                <a href="cart.html" class="icon-btn" style="text-decoration:none;">
                    &#128722; Cart <span class="badge">${cartCount}</span>
                </a>
                ${userSection}
            </div>
        `;
    },

    Hero() {
        return `
            <section class="hero-section">
                <div class="hero-text">
                    <p style="text-transform:uppercase;letter-spacing:5px;color:var(--accent);margin-bottom:1.5rem;">New Summer Collection</p>
                    <h1>STEP INTO<br>THE FUTURE</h1>
                    <a href="catalog.html" class="btn btn-primary" style="text-decoration:none;display:inline-block;">Shop Collection</a>
                </div>
                <div class="hero-visual">
                    <img src="premium_sneaker_1.png" class="hero-shoe" alt="Hero Sneaker">
                </div>
            </section>
        `;
    }
};

window.components = components;

window.deleteReview = async (reviewId, productId) => {
    if (!confirm('Are you sure you want to delete this review?')) return;
    try {
        await api.reviews.delete(reviewId);
        showToast('Review deleted successfully!', 'success');
        
        // Refresh product detail if open
        if (window.openProductDetail) {
            window.openProductDetail(productId);
        }
        
        // Refresh grids to update star counts
        const refreshFuncs = ['renderCatalog', 'renderFeatured', 'loadReviews'];
        for (const funcName of refreshFuncs) {
            if (typeof window[funcName] === 'function') {
                // Determine which cache to update
                let cacheName = funcName === 'renderCatalog' ? 'catalogProducts' : 'featuredCache';
                
                // For renderFeatured, the cache is an object {rows: []}
                const raw = await api.products.list();
                if (raw) {
                    const products = Array.isArray(raw) ? raw : (raw.products || []);
                    for (let prod of products) {
                        try {
                            const res = await api.reviews.listByProduct(prod.id);
                            prod.reviews = Array.isArray(res) ? res : (res && res.reviews ? res.reviews : []);
                        } catch (e) { prod.reviews = []; }
                    }
                    
                    if (funcName === 'renderCatalog' && window.catalogProducts !== undefined) {
                        window.catalogProducts = products;
                    } else if (funcName === 'renderFeatured' && window.featuredCache !== undefined) {
                        window.featuredCache.rows = products.slice(0, 4);
                    }
                    window[funcName]();
                }
            }
        }
    } catch (err) {
        showToast('Failed to delete review: ' + err.message, 'error');
    }
};

window.hasUserReviewed = (product) => {
    if (!state.user || !product.reviews) return false;
    return product.reviews.some(r => r.userId === state.user.id);
};
