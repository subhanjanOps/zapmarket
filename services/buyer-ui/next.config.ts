import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  env: {
    NEXT_PUBLIC_GATEWAY_URL:
      process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8000",
  },
  images: {
    unoptimized: true,
  },
};

export default nextConfig;
