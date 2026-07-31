/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    remotePatterns: [
      { protocol: "http", hostname: "localhost", port: "9000", pathname: "/**" },
    ],
  },
  // Rewrites deep imports from these packages (icons, chart primitives) to
  // their individual source files so webpack only bundles what's actually
  // used, instead of pulling each library's full barrel export.
  experimental: {
    optimizePackageImports: ["lucide-react", "recharts", "@livekit/components-react"],
  },
};

export default nextConfig;
