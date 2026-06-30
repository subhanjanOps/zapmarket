"use client";

import { useEffect } from "react";
import { trackRecentlyViewed } from "./RecentlyViewed";

interface Props {
  productId: string;
  name: string;
  image: string;
}

export function TrackView({ productId, name, image }: Props) {
  useEffect(() => {
    trackRecentlyViewed({ product_id: productId, name, image });
  }, [productId, name, image]);
  return null;
}
