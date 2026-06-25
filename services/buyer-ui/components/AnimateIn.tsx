"use client";
import { useEffect, useRef, useState } from "react";

type AnimationType = "fade-up" | "fade-in" | "scale-in" | "slide-left" | "slide-right" | "slide-up-sm";

interface Props {
  children: React.ReactNode;
  className?: string;
  delay?: number;
  animation?: AnimationType;
  threshold?: number;
}

export default function AnimateIn({
  children,
  className = "",
  delay = 0,
  animation = "fade-up",
  threshold = 0.12,
}: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { threshold }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [threshold]);

  return (
    <div
      ref={ref}
      className={`${className} ${visible ? `animate-${animation}` : "opacity-0"}`}
      style={visible && delay > 0 ? { animationDelay: `${delay}ms` } : undefined}
    >
      {children}
    </div>
  );
}
