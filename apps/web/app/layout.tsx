import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Fynora",
  description: "AI-powered spending intelligence and portfolio planning"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
