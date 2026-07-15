import type { Metadata } from "next";
import { ToastProvider } from "@/components/layout";
import "./globals.css";

export const metadata: Metadata = {
  title: "Clearflow",
  description: "Payment reconciliation and cash-flow intelligence"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body><ToastProvider>{children}</ToastProvider></body>
    </html>
  );
}
