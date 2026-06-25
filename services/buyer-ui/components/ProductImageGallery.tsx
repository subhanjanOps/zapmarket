"use client";
import { useState, useCallback, useEffect } from "react";
import Image from "next/image";
import { ChevronLeft, ChevronRight } from "lucide-react";

interface Props {
  images: string[];
  productName: string;
}

type SlideState = "idle" | "entering-right" | "entering-left" | "exiting-right" | "exiting-left";

const DURATION = 320; // ms

export default function ProductImageGallery({ images, productName }: Props) {
  const list = images.length > 0 ? images : ["/placeholder-product.png"];
  const hasMultiple = list.length > 1;

  const [current, setCurrent] = useState(0);
  const [next, setNext] = useState<number | null>(null);
  const [slideState, setSlideState] = useState<SlideState>("idle");

  const navigate = useCallback(
    (direction: "next" | "prev") => {
      if (slideState !== "idle") return;
      const nextIdx =
        direction === "next"
          ? (current + 1) % list.length
          : (current - 1 + list.length) % list.length;

      setNext(nextIdx);
      setSlideState(direction === "next" ? "entering-right" : "entering-left");
    },
    [current, list.length, slideState]
  );

  // After enter animation plays, commit new index
  useEffect(() => {
    if (slideState === "idle" || next === null) return;
    const t = setTimeout(() => {
      setCurrent(next);
      setNext(null);
      setSlideState("idle");
    }, DURATION);
    return () => clearTimeout(t);
  }, [slideState, next]);

  // Keyboard navigation
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "ArrowRight") navigate("next");
      if (e.key === "ArrowLeft") navigate("prev");
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [navigate]);

  function goTo(i: number) {
    if (i === current || slideState !== "idle") return;
    navigate(i > current ? "next" : "prev");
    // Override index tracking for direct dot/thumb click
    setNext(i);
  }

  // Slide CSS transforms
  const isForward = slideState === "entering-right";
  const currentStyle: React.CSSProperties =
    slideState === "idle"
      ? { transform: "translateX(0)", opacity: 1 }
      : { transform: isForward ? "translateX(-100%)" : "translateX(100%)", opacity: 0.6 };

  const nextStyle: React.CSSProperties =
    next !== null
      ? { transform: "translateX(0)", opacity: 1 }
      : {};

  const nextInitial: React.CSSProperties = {
    transform: isForward ? "translateX(100%)" : "translateX(-100%)",
    opacity: 0.6,
  };

  return (
    <div className="flex flex-col gap-3">
      {/* Main image stage */}
      <div
        className="group aspect-square relative rounded-3xl overflow-hidden"
        style={{
          background: "#FFFCF5",
          border: "1px solid #F0EDE8",
          boxShadow: "0 4px 28px rgba(26,18,8,0.07)",
        }}
      >
        {/* Current image */}
        <div
          className="absolute inset-0"
          style={{
            ...currentStyle,
            transition: slideState !== "idle" ? `transform ${DURATION}ms cubic-bezier(0.4,0,0.2,1), opacity ${DURATION}ms ease` : "none",
          }}
        >
          <Image
            src={list[current]}
            alt={productName}
            fill
            priority
            className="object-contain p-8"
            sizes="(max-width: 768px) 100vw, 50vw"
          />
        </div>

        {/* Incoming image */}
        {next !== null && (
          <div
            className="absolute inset-0"
            style={{
              ...nextInitial,
              ...nextStyle,
              transition: `transform ${DURATION}ms cubic-bezier(0.4,0,0.2,1), opacity ${DURATION}ms ease`,
            }}
          >
            <Image
              src={list[next]}
              alt={productName}
              fill
              className="object-contain p-8"
              sizes="(max-width: 768px) 100vw, 50vw"
            />
          </div>
        )}

        {/* Prev / Next arrows */}
        {hasMultiple && (
          <>
            <button
              onClick={() => navigate("prev")}
              className="absolute left-3 top-1/2 -translate-y-1/2 z-20 w-9 h-9 rounded-full
                         flex items-center justify-center cursor-pointer
                         opacity-0 group-hover:opacity-100 transition-all duration-200
                         hover:scale-110 active:scale-95"
              style={{
                background: "rgba(255,255,255,0.92)",
                boxShadow: "0 2px 8px rgba(26,18,8,0.14)",
              }}
              aria-label="Previous image"
            >
              <ChevronLeft size={18} style={{ color: "#1A1208" }} />
            </button>
            <button
              onClick={() => navigate("next")}
              className="absolute right-3 top-1/2 -translate-y-1/2 z-20 w-9 h-9 rounded-full
                         flex items-center justify-center cursor-pointer
                         opacity-0 group-hover:opacity-100 transition-all duration-200
                         hover:scale-110 active:scale-95"
              style={{
                background: "rgba(255,255,255,0.92)",
                boxShadow: "0 2px 8px rgba(26,18,8,0.14)",
              }}
              aria-label="Next image"
            >
              <ChevronRight size={18} style={{ color: "#1A1208" }} />
            </button>
          </>
        )}

        {/* Dot indicators */}
        {hasMultiple && (
          <div className="absolute bottom-3 left-0 right-0 flex justify-center gap-1.5 z-10">
            {list.map((_, i) => (
              <button
                key={i}
                onClick={() => goTo(i)}
                className="rounded-full cursor-pointer transition-all duration-300"
                style={{
                  width: i === current ? "20px" : "6px",
                  height: "6px",
                  background: i === current ? "#FF2D78" : "rgba(26,18,8,0.18)",
                }}
                aria-label={`Image ${i + 1}`}
              />
            ))}
          </div>
        )}

        {/* Image counter badge */}
        {hasMultiple && (
          <div
            className="absolute top-3 right-3 z-10 text-[10px] font-bold px-2 py-1 rounded-full"
            style={{ background: "rgba(26,18,8,0.55)", color: "#fff" }}
          >
            {current + 1} / {list.length}
          </div>
        )}
      </div>

      {/* Thumbnail strip */}
      {hasMultiple && (
        <div className="flex gap-2 overflow-x-auto pb-1" style={{ scrollbarWidth: "none" }}>
          {list.map((src, i) => (
            <button
              key={i}
              onClick={() => goTo(i)}
              className="shrink-0 w-16 h-16 rounded-xl overflow-hidden cursor-pointer transition-all duration-200 hover:scale-105"
              style={{
                border: i === current ? "2px solid #FF2D78" : "2px solid #F0EDE8",
                background: "#FFFCF5",
                boxShadow: i === current ? "0 0 0 3px rgba(255,45,120,0.15)" : "none",
              }}
              aria-label={`View image ${i + 1}`}
            >
              <div className="relative w-full h-full">
                <Image src={src} alt="" fill className="object-contain p-1" sizes="64px" />
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
