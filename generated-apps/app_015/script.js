// Navbar shadow on scroll
(function(){
  const nav = document.getElementById('mainNavbar');
  const onScroll = () => {
    const scrolled = window.scrollY > 8;
    nav.classList.toggle('shadow-sm', scrolled);
    nav.classList.toggle('bg-body', scrolled);
  };
  onScroll();
  window.addEventListener('scroll', onScroll, {passive:true});
})();

// Smooth scroll for internal anchors
(function(){
  document.querySelectorAll('a[href^="#"]').forEach(a => {
    a.addEventListener('click', (e) => {
      const id = a.getAttribute('href');
      if(id.length > 1){
        const target = document.querySelector(id);
        if(target){
          e.preventDefault();
          target.scrollIntoView({behavior:'smooth'});
        }
      }
    })
  })
})();

// Intersection Observer reveal
(function(){
  const items = document.querySelectorAll('[data-reveal]');
  const io = new IntersectionObserver((entries)=>{
    entries.forEach((entry)=>{
      if(entry.isIntersecting){
        entry.target.classList.add('revealed');
        io.unobserve(entry.target);
      }
    })
  }, {threshold: 0.12});
  items.forEach(el=>io.observe(el));
})();

// Order logic
(function(){
  const paketSelect = document.getElementById('paketSelect');
  const porsiInput = document.getElementById('porsiInput');
  const totalDisplay = document.getElementById('totalDisplay');
  const form = document.getElementById('orderForm');
  const yearSpan = document.getElementById('year');
  if(yearSpan){ yearSpan.textContent = new Date().getFullYear(); }

  function getHargaPaket(){
    const selected = paketSelect.options[paketSelect.selectedIndex];
    const harga = Number(selected.dataset.harga || '0');
    return isFinite(harga) ? harga : 0;
  }

  function formatRupiah(n){
    try{
      return new Intl.NumberFormat('id-ID', {style:'currency', currency:'IDR', maximumFractionDigits:0}).format(n);
    }catch(_){
      return 'Rp' + (n||0).toLocaleString('id-ID');
    }
  }

  function updateTotal(){
    const harga = getHargaPaket();
    const porsi = Math.max(0, Number(porsiInput.value || 0));
    const total = harga * porsi;
    totalDisplay.textContent = formatRupiah(total);
  }

  paketSelect?.addEventListener('change', updateTotal);
  porsiInput?.addEventListener('input', updateTotal);
  updateTotal();

  // Pre-fill from chosen package buttons
  document.querySelectorAll('.btn-pilih-paket').forEach(btn=>{
    btn.addEventListener('click', ()=>{
      const nama = btn.getAttribute('data-paket');
      const harga = btn.getAttribute('data-harga');
      // Set select to matching option
      [...paketSelect.options].forEach(opt=>{
        if(opt.value === nama){
          opt.selected = true;
          opt.dataset.harga = String(harga);
        }
      });
      updateTotal();
      document.getElementById('pesan')?.scrollIntoView({behavior:'smooth'});
    })
  })

  // WhatsApp submit
  form?.addEventListener('submit', (e)=>{
    e.preventDefault();
    const fd = new FormData(form);
    const nama = (fd.get('nama')||'').toString().trim();
    const wa = (fd.get('wa')||'').toString().trim();
    const tanggal = (fd.get('tanggal')||'').toString();
    const jenis = (fd.get('jenis')||'').toString();
    const paket = (fd.get('paket')||'').toString();
    const porsi = Number(fd.get('porsi')||0);
    const catatan = (fd.get('catatan')||'').toString().trim();

    if(!nama || !wa || !tanggal || !jenis || !paket || !porsi){
      alert('Mohon lengkapi semua data pemesanan.');
      return;
    }

    const harga = getHargaPaket();
    const total = harga * porsi;

    const pesan = `Halo UMKM Catering,%0A%0ASaya ingin memesan:%0A- Nama: ${encodeURIComponent(nama)}%0A- WA: ${encodeURIComponent(wa)}%0A- Tanggal: ${encodeURIComponent(tanggal)}%0A- Jenis Acara: ${encodeURIComponent(jenis)}%0A- Paket: ${encodeURIComponent(paket)}%0A- Porsi: ${encodeURIComponent(String(porsi))}%0A- Estimasi Total: ${encodeURIComponent(new Intl.NumberFormat('id-ID',{style:'currency',currency:'IDR',maximumFractionDigits:0}).format(total))}%0A- Catatan: ${encodeURIComponent(catatan || '-')}%0A%0ATerima kasih.`;

    // Ganti nomor berikut dengan nomor admin Anda (format internasional tanpa '+', contoh: 6281234567890)
    const adminNumber = '6281234567890';
    const url = `https://wa.me/${adminNumber}?text=${pesan}`;

    // Open in new tab to avoid navigation issues
    window.open(url, '_blank', 'noopener');
  })
})();
