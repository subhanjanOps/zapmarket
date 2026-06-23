"use client";
import Image from "next/image";
import Link from "next/link";
import { useCartStore } from "@/lib/cart";
import { Minus, Plus, Trash2 } from "lucide-react";

export default function CartPage() {
  const { items, removeItem, updateQty, total } = useCartStore();

  if (items.length === 0) {
    return (
      <div className="max-w-3xl mx-auto px-4 py-20 text-center">
        <h1 className="text-2xl font-bold mb-4">Your cart is empty</h1>
        <Link href="/products" className="bg-[#FF9900] text-black font-semibold px-6 py-3 rounded hover:bg-[#e68900]">
          Start Shopping
        </Link>
      </div>
    );
  }

  const subtotal = total();

  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-6">Shopping Cart</h1>
      <div className="flex flex-col lg:flex-row gap-8">
        <div className="flex-1 space-y-4">
          {items.map((item) => (
            <div key={item.skuId} className="flex gap-4 bg-white border border-gray-200 rounded-lg p-4">
              <div className="w-20 h-20 relative shrink-0 bg-gray-100 rounded">
                <Image src={item.image || "/placeholder-product.png"} alt={item.name} fill className="object-contain p-1" sizes="80px" />
              </div>
              <div className="flex-1">
                <p className="text-sm font-medium text-gray-900">{item.name}</p>
                <p className="text-sm font-bold mt-1">{item.currency} {(item.price / 100).toFixed(2)}</p>
                <div className="flex items-center gap-2 mt-2">
                  <button onClick={() => updateQty(item.skuId, item.qty - 1)} className="p-1 border rounded hover:bg-gray-100"><Minus size={14} /></button>
                  <span className="text-sm w-6 text-center">{item.qty}</span>
                  <button onClick={() => updateQty(item.skuId, item.qty + 1)} className="p-1 border rounded hover:bg-gray-100"><Plus size={14} /></button>
                  <button onClick={() => removeItem(item.skuId)} className="ml-2 p-1 text-red-400 hover:text-red-600"><Trash2 size={14} /></button>
                </div>
              </div>
              <p className="text-sm font-bold shrink-0">{item.currency} {((item.price * item.qty) / 100).toFixed(2)}</p>
            </div>
          ))}
        </div>

        <div className="lg:w-72">
          <div className="bg-white border border-gray-200 rounded-lg p-6 sticky top-20">
            <h2 className="font-semibold mb-4">Order Summary</h2>
            <div className="flex justify-between text-sm mb-2">
              <span>Subtotal</span>
              <span>INR {(subtotal / 100).toFixed(2)}</span>
            </div>
            <div className="flex justify-between text-sm mb-4 text-gray-500">
              <span>Estimated tax</span>
              <span>Included</span>
            </div>
            <div className="flex justify-between font-bold border-t pt-3">
              <span>Total</span>
              <span>INR {(subtotal / 100).toFixed(2)}</span>
            </div>
            <Link href="/checkout" className="mt-4 block w-full text-center bg-[#FF9900] text-black font-semibold py-3 rounded hover:bg-[#e68900]">
              Proceed to Checkout
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
