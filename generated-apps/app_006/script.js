// DOM Elements
const actionBtn = document.getElementById('actionBtn');
const counter = document.getElementById('counter');
const incrementBtn = document.getElementById('incrementBtn');
const decrementBtn = document.getElementById('decrementBtn');
const resetBtn = document.getElementById('resetBtn');

// Counter state
let count = 0;

// Action Button Handler
actionBtn.addEventListener('click', function() {
    // Create a fun animation effect
    const messages = [
        'Keren! 🎉',
        'Bagus! ✨',
        'Mantap! 🚀',
        'Luar Biasa! 💫',
        'Sempurna! ⭐'
    ];
    
    const randomMessage = messages[Math.floor(Math.random() * messages.length)];
    
    // Change button text temporarily
    const originalText = actionBtn.textContent;
    actionBtn.textContent = randomMessage;
    actionBtn.style.transform = 'scale(1.1)';
    
    setTimeout(() => {
        actionBtn.textContent = originalText;
        actionBtn.style.transform = 'scale(1)';
    }, 1000);
    
    // Add ripple effect
    createRipple(event);
});

// Counter Functions
function updateCounter() {
    counter.textContent = count;
    
    // Add animation effect
    counter.classList.add('updated');
    setTimeout(() => {
        counter.classList.remove('updated');
    }, 200);
    
    // Change color based on value
    if (count > 0) {
        counter.style.color = '#667eea';
    } else if (count < 0) {
        counter.style.color = '#ff6b6b';
    } else {
        counter.style.color = '#667eea';
    }
}

incrementBtn.addEventListener('click', function() {
    count++;
    updateCounter();
    animateButton(this);
});

decrementBtn.addEventListener('click', function() {
    count--;
    updateCounter();
    animateButton(this);
});

resetBtn.addEventListener('click', function() {
    count = 0;
    updateCounter();
    animateButton(this);
    
    // Add special effect for reset
    counter.style.transform = 'rotate(360deg)';
    setTimeout(() => {
        counter.style.transform = 'rotate(0deg)';
    }, 500);
});

// Button Animation
function animateButton(button) {
    button.style.transform = 'scale(0.95)';
    setTimeout(() => {
        button.style.transform = 'scale(1)';
    }, 100);
}

// Ripple Effect
function createRipple(event) {
    const button = event.currentTarget;
    const ripple = document.createElement('span');
    const rect = button.getBoundingClientRect();
    const size = Math.max(rect.width, rect.height);
    const x = event.clientX - rect.left - size / 2;
    const y = event.clientY - rect.top - size / 2;
    
    ripple.style.width = ripple.style.height = size + 'px';
    ripple.style.left = x + 'px';
    ripple.style.top = y + 'px';
    ripple.classList.add('ripple');
    
    button.appendChild(ripple);
    
    setTimeout(() => {
        ripple.remove();
    }, 600);
}

// Add ripple CSS dynamically
const style = document.createElement('style');
style.textContent = `
    .btn-primary {
        position: relative;
        overflow: hidden;
    }
    
    .ripple {
        position: absolute;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.6);
        transform: scale(0);
        animation: ripple-animation 0.6s ease-out;
        pointer-events: none;
    }
    
    @keyframes ripple-animation {
        to {
            transform: scale(4);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);

// Initialize
updateCounter();

// Add smooth scroll for any anchor links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    });
});
