<template>
  <div class="sample-content">
    <!-- Header -->
    <header class="site-header">
      <div class="container">
        <h1 class="site-title">TechStore</h1>
        <nav class="main-nav">
          <a href="#" class="nav-link">Home</a>
          <a href="#" class="nav-link">Products</a>
          <a href="#" class="nav-link">About</a>
        </nav>
      </div>
    </header>

    <!-- Banner -->
    <div class="banner">
      <div class="container">
        <h2 class="banner-title">Summer Sale - Up to 50% Off!</h2>
        <p class="banner-subtitle">Limited time offer on selected items</p>
      </div>
    </div>

    <!-- Product Grid -->
    <div class="container">
      <h2 class="section-title">Featured Products</h2>
      
      <div class="product-grid">
        <!-- Product 1 -->
        <article class="product-card" data-product-id="TECH-001">
          <div class="product-badges">
            <span class="badge badge-sale">Sale</span>
            <span class="badge badge-new">New</span>
          </div>
          <img 
            src="https://via.placeholder.com/300x300/3b82f6/ffffff?text=Laptop" 
            alt="Premium Laptop" 
            class="product-image"
            data-full-image="https://via.placeholder.com/1200x1200/3b82f6/ffffff?text=Laptop"
          />
          <div class="product-info">
            <h3 class="product-title">Premium Ultrabook Laptop</h3>
            <p class="product-brand">TechPro</p>
            <div class="product-rating">
              <span class="stars">★★★★★</span>
              <span class="rating-count">(245 reviews)</span>
            </div>
            <p class="product-description">
              Powerful 14-inch laptop with Intel Core i7 processor, 16GB RAM, and 512GB SSD. 
              Perfect for professionals and content creators.
            </p>
            <div class="product-specs">
              <span class="spec">Intel i7</span>
              <span class="spec">16GB RAM</span>
              <span class="spec">512GB SSD</span>
            </div>
            <div class="product-pricing">
              <span class="price-original">$1,299.99</span>
              <span class="price-current">$999.99</span>
              <span class="discount-percent">23% OFF</span>
            </div>
            <div class="product-stock">
              <span class="stock-status in-stock">In Stock</span>
              <span class="stock-quantity">15 available</span>
            </div>
            <button class="btn-add-cart" data-sku="TECH-001">Add to Cart</button>
          </div>
        </article>

        <!-- Product 2 -->
        <article class="product-card" data-product-id="TECH-002">
          <div class="product-badges">
            <span class="badge badge-bestseller">Bestseller</span>
          </div>
          <img 
            src="https://via.placeholder.com/300x300/10b981/ffffff?text=Smartphone" 
            alt="5G Smartphone" 
            class="product-image"
            data-full-image="https://via.placeholder.com/1200x1200/10b981/ffffff?text=Smartphone"
          />
          <div class="product-info">
            <h3 class="product-title">5G Flagship Smartphone</h3>
            <p class="product-brand">PhonePlus</p>
            <div class="product-rating">
              <span class="stars">★★★★☆</span>
              <span class="rating-count">(892 reviews)</span>
            </div>
            <p class="product-description">
              Latest 5G smartphone with 6.7-inch AMOLED display, 128GB storage, 
              and professional-grade triple camera system.
            </p>
            <div class="product-specs">
              <span class="spec">6.7" AMOLED</span>
              <span class="spec">128GB</span>
              <span class="spec">5G</span>
            </div>
            <div class="product-pricing">
              <span class="price-current">$799.99</span>
            </div>
            <div class="product-stock">
              <span class="stock-status in-stock">In Stock</span>
              <span class="stock-quantity">42 available</span>
            </div>
            <button class="btn-add-cart" data-sku="TECH-002">Add to Cart</button>
          </div>
        </article>

        <!-- Product 3 -->
        <article class="product-card" data-product-id="TECH-003">
          <div class="product-badges">
            <span class="badge badge-sale">Sale</span>
          </div>
          <img 
            src="https://via.placeholder.com/300x300/8b5cf6/ffffff?text=Headphones" 
            alt="Wireless Headphones" 
            class="product-image"
            data-full-image="https://via.placeholder.com/1200x1200/8b5cf6/ffffff?text=Headphones"
          />
          <div class="product-info">
            <h3 class="product-title">Noise-Canceling Wireless Headphones</h3>
            <p class="product-brand">AudioMax</p>
            <div class="product-rating">
              <span class="stars">★★★★★</span>
              <span class="rating-count">(1,547 reviews)</span>
            </div>
            <p class="product-description">
              Premium over-ear headphones with active noise cancellation, 
              30-hour battery life, and studio-quality sound.
            </p>
            <div class="product-specs">
              <span class="spec">ANC</span>
              <span class="spec">30h Battery</span>
              <span class="spec">Bluetooth 5.2</span>
            </div>
            <div class="product-pricing">
              <span class="price-original">$349.99</span>
              <span class="price-current">$279.99</span>
              <span class="discount-percent">20% OFF</span>
            </div>
            <div class="product-stock">
              <span class="stock-status in-stock">In Stock</span>
              <span class="stock-quantity">28 available</span>
            </div>
            <button class="btn-add-cart" data-sku="TECH-003">Add to Cart</button>
          </div>
        </article>

        <!-- Product 4 -->
        <article class="product-card" data-product-id="TECH-004">
          <div class="product-badges">
            <span class="badge badge-limited">Limited</span>
          </div>
          <img 
            src="https://via.placeholder.com/300x300/ef4444/ffffff?text=Smartwatch" 
            alt="Fitness Smartwatch" 
            class="product-image"
            data-full-image="https://via.placeholder.com/1200x1200/ef4444/ffffff?text=Smartwatch"
          />
          <div class="product-info">
            <h3 class="product-title">Advanced Fitness Smartwatch</h3>
            <p class="product-brand">WearTech</p>
            <div class="product-rating">
              <span class="stars">★★★★☆</span>
              <span class="rating-count">(634 reviews)</span>
            </div>
            <p class="product-description">
              Track your health and fitness with GPS, heart rate monitoring, 
              sleep tracking, and 50+ sport modes.
            </p>
            <div class="product-specs">
              <span class="spec">GPS</span>
              <span class="spec">Heart Rate</span>
              <span class="spec">Waterproof</span>
            </div>
            <div class="product-pricing">
              <span class="price-current">$249.99</span>
            </div>
            <div class="product-stock">
              <span class="stock-status low-stock">Low Stock</span>
              <span class="stock-quantity">3 available</span>
            </div>
            <button class="btn-add-cart" data-sku="TECH-004">Add to Cart</button>
          </div>
        </article>
      </div>
    </div>

    <!-- Footer -->
    <footer class="site-footer">
      <div class="container">
        <p class="footer-text">© 2024 TechStore. All rights reserved.</p>
        <div class="footer-links">
          <a href="#" class="footer-link">Privacy Policy</a>
          <a href="#" class="footer-link">Terms of Service</a>
          <a href="#" class="footer-link">Contact Us</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<style scoped>
/* Sample Content Styles */
.sample-content {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  min-height: 100vh;
  background: #f9fafb;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 20px;
}

/* Header */
.site-header {
  background: white;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  padding: 20px 0;
  margin-bottom: 20px;
}

.site-header .container {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.site-title {
  font-size: 28px;
  font-weight: bold;
  color: #1f2937;
  margin: 0;
}

.main-nav {
  display: flex;
  gap: 24px;
}

.nav-link {
  color: #6b7280;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s;
}

.nav-link:hover {
  color: #3b82f6;
}

/* Banner */
.banner {
  background: linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%);
  color: white;
  padding: 40px 0;
  margin-bottom: 40px;
  text-align: center;
}

.banner-title {
  font-size: 36px;
  font-weight: bold;
  margin: 0 0 10px 0;
}

.banner-subtitle {
  font-size: 18px;
  opacity: 0.9;
  margin: 0;
}

/* Product Grid */
.section-title {
  font-size: 32px;
  font-weight: bold;
  color: #1f2937;
  margin: 0 0 30px 0;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 24px;
  margin-bottom: 60px;
}

.product-card {
  background: white;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  transition: transform 0.3s, box-shadow 0.3s;
  position: relative;
}

.product-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0,0,0,0.15);
}

.product-badges {
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: bold;
  text-transform: uppercase;
}

.badge-sale { background: #ef4444; color: white; }
.badge-new { background: #10b981; color: white; }
.badge-bestseller { background: #f59e0b; color: white; }
.badge-limited { background: #8b5cf6; color: white; }

.product-image {
  width: 100%;
  height: 280px;
  object-fit: cover;
}

.product-info {
  padding: 20px;
}

.product-title {
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
  margin: 0 0 6px 0;
  line-height: 1.4;
}

.product-brand {
  font-size: 14px;
  color: #6b7280;
  margin: 0 0 8px 0;
}

.product-rating {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
}

.stars {
  color: #f59e0b;
  font-size: 14px;
}

.rating-count {
  font-size: 12px;
  color: #9ca3af;
}

.product-description {
  font-size: 14px;
  color: #4b5563;
  line-height: 1.6;
  margin: 0 0 12px 0;
}

.product-specs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.spec {
  background: #f3f4f6;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  color: #6b7280;
  font-weight: 500;
}

.product-pricing {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.price-original {
  text-decoration: line-through;
  color: #9ca3af;
  font-size: 14px;
}

.price-current {
  font-size: 24px;
  font-weight: bold;
  color: #1f2937;
}

.discount-percent {
  background: #dcfce7;
  color: #16a34a;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
}

.product-stock {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.stock-status {
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
}

.stock-status.in-stock {
  background: #dcfce7;
  color: #16a34a;
}

.stock-status.low-stock {
  background: #fef3c7;
  color: #d97706;
}

.stock-quantity {
  font-size: 12px;
  color: #6b7280;
}

.btn-add-cart {
  width: 100%;
  background: #3b82f6;
  color: white;
  border: none;
  padding: 12px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-add-cart:hover {
  background: #2563eb;
}

/* Footer */
.site-footer {
  background: #1f2937;
  color: white;
  padding: 40px 0;
  margin-top: 60px;
}

.site-footer .container {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-text {
  margin: 0;
  color: #9ca3af;
}

.footer-links {
  display: flex;
  gap: 20px;
}

.footer-link {
  color: #9ca3af;
  text-decoration: none;
  font-size: 14px;
  transition: color 0.2s;
}

.footer-link:hover {
  color: white;
}
</style>
