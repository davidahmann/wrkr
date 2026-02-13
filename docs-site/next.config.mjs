/** @type {import('next').NextConfig} */
const basePath = process.env.DOCS_SITE_BASE_PATH || ''

const nextConfig = {
  output: 'export',
  trailingSlash: true,
  basePath,
  assetPrefix: basePath || undefined,
  images: {
    unoptimized: true,
  },
}

export default nextConfig
