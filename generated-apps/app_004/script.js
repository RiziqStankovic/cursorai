// Mobile Navigation Toggle
const navToggle = document.querySelector('.nav-toggle');
const navMenu = document.querySelector('.nav-menu');

if (navToggle) {
    navToggle.addEventListener('click', () => {
        navMenu.classList.toggle('active');
    });
}

// Close mobile menu when clicking on a link
const navLinks = document.querySelectorAll('.nav-menu a');
navLinks.forEach(link => {
    link.addEventListener('click', () => {
        navMenu.classList.remove('active');
    });
});

// Smooth Scrolling
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

// Investment Calculator
const calculateBtn = document.getElementById('calculate-btn');
const initialAmountInput = document.getElementById('initial-amount');
const monthlyAmountInput = document.getElementById('monthly-amount');
const returnRateInput = document.getElementById('return-rate');
const yearsInput = document.getElementById('years');

const totalInvestedEl = document.getElementById('total-invested');
const finalValueEl = document.getElementById('final-value');
const profitEl = document.getElementById('profit');

function formatCurrency(amount) {
    return new Intl.NumberFormat('id-ID', {
        style: 'currency',
        currency: 'IDR',
        minimumFractionDigits: 0,
        maximumFractionDigits: 0
    }).format(amount);
}

function calculateInvestment() {
    const initialAmount = parseFloat(initialAmountInput.value) || 0;
    const monthlyAmount = parseFloat(monthlyAmountInput.value) || 0;
    const returnRate = parseFloat(returnRateInput.value) || 0;
    const years = parseFloat(yearsInput.value) || 0;

    if (years <= 0) {
        alert('Jangka waktu harus lebih dari 0 tahun');
        return;
    }

    // Convert annual return rate to monthly
    const monthlyRate = returnRate / 100 / 12;
    const months = years * 12;

    // Calculate future value of initial investment
    const futureValueInitial = initialAmount * Math.pow(1 + monthlyRate, months);

    // Calculate future value of monthly contributions
    // FV = PMT * [((1 + r)^n - 1) / r]
    let futureValueMonthly = 0;
    if (monthlyRate > 0) {
        futureValueMonthly = monthlyAmount * ((Math.pow(1 + monthlyRate, months) - 1) / monthlyRate);
    } else {
        futureValueMonthly = monthlyAmount * months;
    }

    // Total future value
    const finalValue = futureValueInitial + futureValueMonthly;

    // Total invested
    const totalInvested = initialAmount + (monthlyAmount * months);

    // Profit
    const profit = finalValue - totalInvested;

    // Update UI
    totalInvestedEl.textContent = formatCurrency(totalInvested);
    finalValueEl.textContent = formatCurrency(finalValue);
    profitEl.textContent = formatCurrency(profit);

    // Add animation
    [totalInvestedEl, finalValueEl, profitEl].forEach(el => {
        el.style.transform = 'scale(1.1)';
        setTimeout(() => {
            el.style.transform = 'scale(1)';
        }, 300);
    });
}

if (calculateBtn) {
    calculateBtn.addEventListener('click', calculateInvestment);
}

// Allow Enter key to trigger calculation
[initialAmountInput, monthlyAmountInput, returnRateInput, yearsInput].forEach(input => {
    if (input) {
        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                calculateInvestment();
            }
        });
    }
});

// Add scroll animation on scroll
const observerOptions = {
    threshold: 0.1,
    rootMargin: '0px 0px -50px 0px'
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.style.opacity = '1';
            entry.target.style.transform = 'translateY(0)';
        }
    });
}, observerOptions);

// Observe elements for animation
document.addEventListener('DOMContentLoaded', () => {
    const animateElements = document.querySelectorAll('.type-card, .tip-card, .calculator-form, .result-card');
    animateElements.forEach(el => {
        el.style.opacity = '0';
        el.style.transform = 'translateY(20px)';
        el.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
        observer.observe(el);
    });
});

// Add hover effect to cards
document.querySelectorAll('.type-card, .tip-card').forEach(card => {
    card.addEventListener('mouseenter', function() {
        this.style.transition = 'all 0.3s ease';
    });
});

// Format input numbers as user types (optional enhancement)
function formatNumberInput(input) {
    input.addEventListener('input', function() {
        // Remove any non-numeric characters except decimal point
        this.value = this.value.replace(/[^0-9.]/g, '');
    });
}

if (initialAmountInput) formatNumberInput(initialAmountInput);
if (monthlyAmountInput) formatNumberInput(monthlyAmountInput);
if (returnRateInput) formatNumberInput(returnRateInput);
if (yearsInput) formatNumberInput(yearsInput);
