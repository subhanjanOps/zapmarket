"use client";

import { motion, type Variants, type HTMLMotionProps } from "framer-motion";

type Variant = "fade-up" | "fade-in" | "slide-left" | "slide-right" | "scale-in" | "stagger-child";

const VARIANTS: Record<Variant, Variants> = {
  "fade-up": {
    hidden:  { opacity: 0, y: 32 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.6, ease: [0.16, 1, 0.3, 1] } },
  },
  "fade-in": {
    hidden:  { opacity: 0 },
    visible: { opacity: 1, transition: { duration: 0.4, ease: "easeOut" } },
  },
  "slide-left": {
    hidden:  { opacity: 0, x: -28 },
    visible: { opacity: 1, x: 0, transition: { duration: 0.5, ease: [0.16, 1, 0.3, 1] } },
  },
  "slide-right": {
    hidden:  { opacity: 0, x: 28 },
    visible: { opacity: 1, x: 0, transition: { duration: 0.5, ease: [0.16, 1, 0.3, 1] } },
  },
  "scale-in": {
    hidden:  { opacity: 0, scale: 0.92 },
    visible: { opacity: 1, scale: 1, transition: { duration: 0.5, ease: [0.16, 1, 0.3, 1] } },
  },
  "stagger-child": {
    hidden:  { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.45, ease: [0.16, 1, 0.3, 1] } },
  },
};

const STAGGER_CONTAINER: Variants = {
  hidden:  {},
  visible: { transition: { staggerChildren: 0.07, delayChildren: 0.1 } },
};

interface MotionWrapperProps extends Omit<HTMLMotionProps<"div">, "variants"> {
  variant?: Variant;
  delay?: number;
  stagger?: boolean;
  className?: string;
  children: React.ReactNode;
}

export function MotionWrapper({
  variant = "fade-up",
  delay = 0,
  stagger = false,
  className,
  children,
  ...props
}: MotionWrapperProps) {
  if (stagger) {
    return (
      <motion.div
        className={className}
        variants={STAGGER_CONTAINER}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true, margin: "-60px" }}
        {...props}
      >
        {children}
      </motion.div>
    );
  }

  return (
    <motion.div
      className={className}
      variants={VARIANTS[variant]}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-60px" }}
      transition={{ delay }}
      {...props}
    >
      {children}
    </motion.div>
  );
}

export function MotionChild({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <motion.div className={className} variants={VARIANTS["stagger-child"]}>
      {children}
    </motion.div>
  );
}
