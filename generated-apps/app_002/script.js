// Product Data
const products = [
    {
        id: 1,
        name: "Espresso",
        description: "Kopi hitam pekat dengan rasa yang kuat dan kaya",
        price: 25000,
        category: "hot",
        emoji: "?"
    },
    {
        id: 2,
        name: "Cappuccino",
        description: "Espresso dengan susu steamed dan busa susu yang lembut",
        price: 30000,
        category: "hot",
        emoji: "?"
    },
    {
        id: 3,
        name: "Latte",
        description: "Espresso dengan susu steamed dalam porsi yang lebih banyak",
        price: 32000,
        category: "hot",
        emoji: "?"
    },
    {
        id: 4,
        name: "Americano",
        description: "Espresso dengan air panas, rasa kopi yang halus",
        price: 28000,
        category: "hot",
        emoji: "?"
    },
    {
        id: 5,
        name: "Mocha",
        description: "Espresso dengan coklat dan susu steamed",
        price: 35000,
        category: "hot",
        emoji: "?"
    },
    {
        id: 6,
        name: "Iced Coffee",
        description: "Kopi dingin dengan es batu, menyegarkan",
        price: 28000,
        category: "cold",
        emoji: "??"
    },
    {
        id: 7,
        name: "Cold Brew",
        description: "Kopi yang diseduh dingin selama 12 jam, rasa yang halus",
        price: 30000,
        category: "cold",
        emoji: "??"
    },
    {
        id: 8,
        name: "Frappuccino",
        description: "Minuman kopi dingin dengan es blended",
        price: 38000,
        category: "cold",
        emoji: "??"
    },
    {
        id: 9,
        name: "Iced Latte",
        description: "Latte dingin dengan es batu",
        price: 33000,
        category: "cold",
        emoji: "??"
    },
    {
        id: 10,
        name: "Croissant",
        description: "Pastry Prancis yang renyah dan buttery",
        price: 25000,
        category: "snack",
        emoji: "??"
    },
    {
        id: 11,
        name: "Muffin Coklat",
        description: "Muffin lembut dengan coklat chips",
        price: 20000,
        category: "snack",
        emoji: "??"
    },
    {
        id: 12,
        name: "Donat",
        description: "Donat manis dengan berbagai topping",
        price: 15000,
        category: "snack",
        emoji: "??"
    }
];

// Shopping Cart
let cart = [];
let currentFilter = 'all';

// DOM Elements
const productsGrid = document.getElementById('productsGrid');
const cartBtn = document.getElementById('cartBtn');
const cartOverlay = document.getElementById('cartOverlay');
const cartSidebar = document.getElementById('cartSidebar');
const closeCart = document.getElementById('closeCart');
const cartItems = document.getElementById('cartItems');
const cartCount = document.getElementById('cartCount');
const cartTotal = document.getElementById('cartTotal');
const checkoutBtn = document.getElementById('checkoutBtn');
const filterButtons = document.querySelectorAll('.filter-btn');
const navLinks = document.querySelectorAll('.nav-link');
const menuToggle = document.getElementById('menuToggle');
const navMenu = document.getElementById('navMenu');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    renderProducts();
    updateCartUI();
    setupEventListeners();
});

// Setup Event Listeners
function setupEventListeners() {
    // Cart toggle
    cartBtn.addEventListener('click', openCart);
    closeCart.addEventListener('click', closeCartSidebar);
    cartOverlay.addEventListener('click', closeCartSidebar);
    checkoutBtn.addEventListener('click', handleCheckout);

    // Filter buttons
    filterButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            filterButtons.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            currentFilter = btn.dataset.filter;
            renderProducts();
        });
    });

    // Smooth scrolling for navigation links
    navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const targetId = link.getAttribute('href');
            const targetSection = document.querySelector(targetId);
            
            if (targetSection) {
                targetSection.scrollIntoView({ behavior: 'smooth' });
                updateActiveNavLink(link);
                
                // Close mobile menu if open
                navMenu.classList.remove('active');
            }
        });
    });

    // Mobile menu toggle
    menuToggle.addEventListener('click', () => {
        navMenu.classList.toggle('active');
    });

    // Update active nav link on scroll
    window.addEventListener('scroll', updateNavOnScroll);

    // Close cart on Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && cartSidebar.classList.contains('active')) {
            closeCartSidebar();
        }
    });
}

// Render Products
function renderProducts() {
    const filteredProducts = currentFilter === 'all' 
        ? products 
        : products.filter(p => p.category === currentFilter);

    productsGrid.innerHTML = filteredProducts.map(product => `
        <div class="product-card" data-category="${product.category}">
            <div class="product-image">
                <span style="font-size: 4rem;">${product.emoji}</span>
            </div>
            <div class="product-info">
                <h3 class="product-name">${product.name}</h3>
                <p class="product-description">${product.description}</p>
                <div class="product-footer">
                    <span class="product-price">${formatPrice(product.price)}</span>
                    <button class="add-to-cart" onclick="addToCart(${product.id})">
                        <i class="fas fa-plus"></i> Tambah
                    </button>
                </div>
            </div>
        </div>
    `).join('');
}

// Add to Cart
function addToCart(productId) {
    const product = products.find(p => p.id === productId);
    if (!product) return;

    const existingItem = cart.find(item => item.id === productId);
    
    if (existingItem) {
        existingItem.quantity++;
    } else {
        cart.push({
            ...product,
            quantity: 1
        });
    }

    updateCartUI();
    showNotification(`${product.name} ditambahkan ke keranjang!`);
}

// Remove from Cart
function removeFromCart(productId) {
    cart = cart.filter(item => item.id !== productId);
    updateCartUI();
}

// Update Quantity
function updateQuantity(productId, change) {
    const item = cart.find(item => item.id === productId);
    if (!item) return;

    item.quantity += change;
    
    if (item.quantity <= 0) {
        removeFromCart(productId);
    } else {
        updateCartUI();
    }
}

// Update Cart UI
function updateCartUI() {
    // Update cart count
    const totalItems = cart.reduce((sum, item) => sum + item.quantity, 0);
    cartCount.textContent = totalItems;
    cartCount.style.display = totalItems > 0 ? 'flex' : 'none';

    // Update cart items
    if (cart.length === 0) {
        cartItems.innerHTML = '<div class="empty-cart"><p>Keranjang Anda kosong</p></div>';
    } else {
        cartItems.innerHTML = cart.map(item => `
            <div class="cart-item">
                <div class="cart-item-image">
                    <span>${item.emoji}</span>
                </div>
                <div class="cart-item-info">
                    <div class="cart-item-name">${item.name}</div>
                    <div class="cart-item-price">${formatPrice(item.price)}</div>
                    <div class="cart-item-quantity">
                        <button class="quantity-btn" onclick="updateQuantity(${item.id}, -1)">-</button>
                        <span class="quantity-value">${item.quantity}</span>
                        <button class="quantity-btn" onclick="updateQuantity(${item.id}, 1)">+</button>
                    </div>
                </div>
                <button class="remove-item" onclick="removeFromCart(${item.id})" title="Hapus">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        `).join('');
    }

    // Update total
    const total = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
    cartTotal.textContent = formatPrice(total);
}

// Open Cart
function openCart() {
    cartSidebar.classList.add('active');
    cartOverlay.classList.add('active');
    document.body.style.overflow = 'hidden';
}

// Close Cart
function closeCartSidebar() {
    cartSidebar.classList.remove('active');
    cartOverlay.classList.remove('active');
    document.body.style.overflow = '';
}

// Handle Checkout
function handleCheckout() {
    if (cart.length === 0) {
        showNotification('Keranjang Anda kosong!', 'error');
        return;
    }

    const total = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
    const message = `Terima kasih! Total pembayaran Anda: ${formatPrice(total)}\n\nPesanan Anda sedang diproses.`;
    
    alert(message);
    cart = [];
    updateCartUI();
    closeCartSidebar();
    showNotification('Pesanan berhasil!', 'success');
}

// Format Price
function formatPrice(price) {
    return new Intl.NumberFormat('id-ID', {
        style: 'currency',
        currency: 'IDR',
        minimumFractionDigits: 0
    }).format(price);
}

// Show Notification
function showNotification(message, type = 'success') {
    // Remove existing notification
    const existing = document.querySelector('.notification');
    if (existing) existing.remove();

    // Create notification
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 100px;
        right: 20px;
        background: ${type === 'success' ? '#28a745' : '#dc3545'};
        color: white;
        padding: 1rem 1.5rem;
        border-radius: 5px;
        box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        z-index: 3000;
        animation: slideIn 0.3s ease;
    `;

    document.body.appendChild(notification);

    // Remove after 3 seconds
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Update Active Nav Link
function updateActiveNavLink(activeLink) {
    navLinks.forEach(link => link.classList.remove('active'));
    activeLink.classList.add('active');
}

// Update Nav on Scroll
function updateNavOnScroll() {
    const sections = document.querySelectorAll('section[id]');
    const scrollY = window.pageYOffset;

    sections.forEach(section => {
        const sectionHeight = section.offsetHeight;
        const sectionTop = section.offsetTop - 100;
        const sectionId = section.getAttribute('id');

        if (scrollY > sectionTop && scrollY <= sectionTop + sectionHeight) {
            navLinks.forEach(link => {
                link.classList.remove('active');
                if (link.getAttribute('href') === `#${sectionId}`) {
                    link.classList.add('active');
                }
            });
        }
    });
}

// Add CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(400px);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(400px);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);
