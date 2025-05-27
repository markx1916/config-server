# Stage 1: Build the Vue.js application
FROM node:18-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package.json and package-lock.json (or yarn.lock)
COPY package.json package-lock.json* ./
# If you use yarn, copy yarn.lock instead
# COPY package.json yarn.lock ./

# Install dependencies
RUN npm install
# If you use yarn, use: RUN yarn install --frozen-lockfile

# Copy all source files
COPY . .

# Build the application
RUN npm run build

# Stage 2: Serve the application with Nginx
FROM nginx:stable-alpine

# Remove default Nginx static assets
RUN rm -rf /usr/share/nginx/html/*

# Copy built assets from builder stage to Nginx html directory
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy custom Nginx configuration (if needed)
# For SPA, we need to configure Nginx to redirect all non-asset requests to index.html
# Create a nginx.conf file in the frontend-ui directory or use a default one here.
# Example: COPY nginx.conf /etc/nginx/conf.d/default.conf
# If no custom nginx.conf is provided, Nginx will use its default behavior,
# which might not work well for SPAs (requires rewrite for deep links).

# Create a simple Nginx config for SPA
RUN echo "server { \
    listen 80; \
    server_name localhost; \
    root /usr/share/nginx/html; \
    index index.html index.htm; \
    location / { \
        try_files \$uri \$uri/ /index.html; \
    } \
    location /api { \
        # This proxy_pass is a placeholder. In a real deployment, \
        # this Nginx instance might not be co-located with the backend. \
        # If it is, you'd proxy to the backend service. \
        # If frontend and backend are served by different domains/ports, \
        # this /api block might not be needed here, as CORS would be handled. \
        # For same-origin deployment (frontend and backend proxied by the same Nginx), \
        # this would point to your backend API upstream. \
        # proxy_pass http://backend-service-host:8080/api; # Example \
        # For now, just return 404 if /api is hit directly on this Nginx instance, \
        # assuming API calls are handled by a different server or a higher-level proxy. \
        return 404; \
    } \
    # Optional: Add gzip compression for static assets
    gzip on; \
    gzip_vary on; \
    gzip_proxied any; \
    gzip_comp_level 6; \
    gzip_buffers 16 8k; \
    gzip_http_version 1.1; \
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript image/svg+xml; \
}" > /etc/nginx/conf.d/default.conf

# Expose port 80
EXPOSE 80

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]
