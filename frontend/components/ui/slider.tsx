"use client"

import * as React from "react"
import { cn } from "@/lib/utils"

interface SliderProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'defaultValue' | 'value'> {
  defaultValue?: number[]
  value?: number[]
  onValueChange?: (value: number[]) => void
}

const Slider = React.forwardRef<HTMLInputElement, SliderProps>(
  ({ className, defaultValue, value, onValueChange, ...props }, ref) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      if (onValueChange) {
        onValueChange([parseInt(e.target.value, 10)])
      }
    }

    return (
      <div className={cn("relative flex w-full touch-none select-none items-center group", className)}>
        <input
          type="range"
          ref={ref}
          className={cn(
            "w-full h-1.5 bg-slate-200 dark:bg-slate-800 rounded-lg appearance-none cursor-pointer",
            "accent-brand-600 hover:accent-brand-500 transition-all",
            /* Webkit (Chrome, Safari, Edge) */
            "[&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:h-4 [&::-webkit-slider-thumb]:w-4",
            "[&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-white [&::-webkit-slider-thumb]:border-2",
            "[&::-webkit-slider-thumb]:border-brand-600 [&::-webkit-slider-thumb]:shadow-md [&::-webkit-slider-thumb]:transition-transform",
            "group-hover:[&::-webkit-slider-thumb]:scale-110",
            /* Firefox */
            "[&::-moz-range-thumb]:h-4 [&::-moz-range-thumb]:w-4 [&::-moz-range-thumb]:rounded-full",
            "[&::-moz-range-thumb]:bg-white [&::-moz-range-thumb]:border-2 [&::-moz-range-thumb]:border-brand-600",
            "[&::-moz-range-thumb]:shadow-md [&::-moz-range-thumb]:transition-transform"
          )}
          defaultValue={defaultValue?.[0]}
          value={value?.[0]}
          onChange={handleChange}
          {...props}
        />
      </div>
    )
  }
)
Slider.displayName = "Slider"

export { Slider }
export default Slider
