// Course Data
const coursesData = [
    {
        id: 1,
        title: "Belajar Dasar Pemrograman Web",
        description: "Pelajari HTML, CSS, dan JavaScript dari dasar hingga mahir. Bangun website modern dan responsif.",
        level: "beginner",
        price: "Gratis",
        isFree: true,
        students: 12500,
        rating: 4.8,
        icon: "🌐"
    },
    {
        id: 2,
        title: "React untuk Pemula",
        description: "Kuasai React.js dan buat aplikasi web interaktif dengan komponen yang reusable.",
        level: "beginner",
        price: "Rp 299.000",
        isFree: false,
        students: 8500,
        rating: 4.9,
        icon: "⚛️"
    },
    {
        id: 3,
        title: "Node.js Backend Development",
        description: "Bangun RESTful API dan server-side application dengan Node.js dan Express.",
        level: "intermediate",
        price: "Rp 399.000",
        isFree: false,
        students: 6200,
        rating: 4.7,
        icon: "🚀"
    },
    {
        id: 4,
        title: "Full Stack JavaScript",
        description: "Menjadi full stack developer dengan menguasai frontend dan backend JavaScript.",
        level: "intermediate",
        price: "Rp 599.000",
        isFree: false,
        students: 4800,
        rating: 4.9,
        icon: "💻"
    },
    {
        id: 5,
        title: "Machine Learning dengan Python",
        description: "Pelajari machine learning dari dasar menggunakan Python, TensorFlow, dan scikit-learn.",
        level: "advanced",
        price: "Rp 799.000",
        isFree: false,
        students: 3200,
        rating: 4.8,
        icon: "🤖"
    },
    {
        id: 6,
        title: "Android Development dengan Kotlin",
        description: "Buat aplikasi Android modern menggunakan Kotlin dan Android Jetpack.",
        level: "intermediate",
        price: "Rp 449.000",
        isFree: false,
        students: 5600,
        rating: 4.6,
        icon: "📱"
    },
    {
        id: 7,
        title: "Flutter untuk Mobile Development",
        description: "Kembangkan aplikasi mobile cross-platform dengan Flutter dan Dart.",
        level: "intermediate",
        price: "Rp 499.000",
        isFree: false,
        students: 4100,
        rating: 4.7,
        icon: "🎯"
    },
    {
        id: 8,
        title: "AWS Cloud Practitioner",
        description: "Pelajari cloud computing dengan Amazon Web Services dari dasar hingga advanced.",
        level: "advanced",
        price: "Rp 699.000",
        isFree: false,
        students: 2800,
        rating: 4.9,
        icon: "☁️"
    },
    {
        id: 9,
        title: "UI/UX Design dengan Figma",
        description: "Desain interface yang user-friendly dengan Figma dan prinsip UX design.",
        level: "beginner",
        price: "Rp 349.000",
        isFree: false,
        students: 7200,
        rating: 4.8,
        icon: "🎨"
    }
];

// DOM Elements
const coursesGrid = document.getElementById('coursesGrid');
const filterButtons = document.querySelectorAll('.filter-btn');
const hamburger = document.getElementById('hamburger');
const navMenu = document.getElementById('navMenu');
const navLinks = document.querySelectorAll('.nav-link');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    renderCourses('all');
    setupEventListeners();
    setupSmoothScroll();
});

// Render Courses
function renderCourses(filter) {
    coursesGrid.innerHTML = '';
    
    const filteredCourses = filter === 'all' 
        ? coursesData 
        : coursesData.filter(course => course.level === filter);
    
    filteredCourses.forEach(course => {
        const courseCard = createCourseCard(course);
        coursesGrid.appendChild(courseCard);
    });
}

// Create Course Card
function createCourseCard(course) {
    const card = document.createElement('div');
    card.className = 'course-card';
    card.setAttribute('data-level', course.level);
    
    const levelLabels = {
        beginner: 'Pemula',
        intermediate: 'Menengah',
        advanced: 'Lanjutan'
    };
    
    card.innerHTML = `
        <div class="course-image">
            ${course.icon}
            <span class="course-badge">⭐ ${course.rating}</span>
        </div>
        <div class="course-content">
            <span class="course-level ${course.level}">${levelLabels[course.level]}</span>
            <h3 class="course-title">${course.title}</h3>
            <p class="course-description">${course.description}</p>
            <div class="course-meta">
                <span>👥 ${course.students.toLocaleString('id-ID')} siswa</span>
            </div>
            <div class="course-footer">
                <div class="course-price ${course.isFree ? 'free' : ''}">${course.price}</div>
                <button class="course-btn" onclick="handleEnroll(${course.id})">Daftar</button>
            </div>
        </div>
    `;
    
    return card;
}

// Filter Courses
function setupEventListeners() {
    filterButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            // Remove active class from all buttons
            filterButtons.forEach(b => b.classList.remove('active'));
            // Add active class to clicked button
            btn.classList.add('active');
            // Filter courses
            const filter = btn.getAttribute('data-filter');
            renderCourses(filter);
        });
    });
    
    // Mobile menu toggle
    hamburger.addEventListener('click', () => {
        hamburger.classList.toggle('active');
        navMenu.classList.toggle('active');
    });
    
    // Close mobile menu when clicking on a link
    navLinks.forEach(link => {
        link.addEventListener('click', () => {
            hamburger.classList.remove('active');
            navMenu.classList.remove('active');
        });
    });
    
    // Close mobile menu when clicking outside
    document.addEventListener('click', (e) => {
        if (!hamburger.contains(e.target) && !navMenu.contains(e.target)) {
            hamburger.classList.remove('active');
            navMenu.classList.remove('active');
        }
    });
}

// Smooth Scroll
function setupSmoothScroll() {
    navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            const href = link.getAttribute('href');
            if (href.startsWith('#')) {
                e.preventDefault();
                const targetId = href.substring(1);
                const targetElement = document.getElementById(targetId);
                if (targetElement) {
                    targetElement.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
            }
        });
    });
}

// Handle Enroll
function handleEnroll(courseId) {
    const course = coursesData.find(c => c.id === courseId);
    if (course) {
        alert(`Terima kasih! Anda akan mendaftar ke kursus: "${course.title}"\n\nHarga: ${course.price}\n\nFitur ini akan diarahkan ke halaman pembayaran.`);
    }
}

// Navbar scroll effect
let lastScroll = 0;
window.addEventListener('scroll', () => {
    const navbar = document.querySelector('.navbar');
    const currentScroll = window.pageYOffset;
    
    if (currentScroll > 100) {
        navbar.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.1)';
    } else {
        navbar.style.boxShadow = '0 2px 4px rgba(0, 0, 0, 0.05)';
    }
    
    lastScroll = currentScroll;
});

// Active nav link on scroll
const sections = document.querySelectorAll('section[id]');
window.addEventListener('scroll', () => {
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
});

// Intersection Observer for animations
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
    const animateElements = document.querySelectorAll('.category-card, .feature-card, .testimonial-card');
    animateElements.forEach(el => {
        el.style.opacity = '0';
        el.style.transform = 'translateY(20px)';
        el.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
        observer.observe(el);
    });
});

// Add loading state to buttons
document.querySelectorAll('.btn, .course-btn').forEach(btn => {
    btn.addEventListener('click', function(e) {
        if (!this.classList.contains('nav-link')) {
            const originalText = this.textContent;
            this.textContent = 'Memuat...';
            this.disabled = true;
            
            setTimeout(() => {
                this.textContent = originalText;
                this.disabled = false;
            }, 1000);
        }
    });
});

// Format numbers with Indonesian locale
function formatNumber(num) {
    return num.toLocaleString('id-ID');
}

// Console welcome message
console.log('%c🎓 Selamat datang di BelajarCoding!', 'color: #00D4FF; font-size: 20px; font-weight: bold;');
console.log('%cPlatform pembelajaran teknologi terdepan', 'color: #666; font-size: 14px;');
