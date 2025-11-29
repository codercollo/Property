import type { ReactNode } from "react";

export function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-neutral-50">
      <header className="sticky top-0 z-50 border-b border-neutral-200 bg-white shadow-sm">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary-600">
                <span className="text-sm font-bold text-white">P</span>
              </div>
              <div>
                <h1 className="text-base font-semibold text-neutral-900">PropertyList</h1>
                <p className="text-xs text-neutral-500">Find your perfect home</p>
              </div>
            </div>
            
            <nav className="hidden md:flex items-center gap-1">
              <a 
                href="#" 
                className="rounded-md px-3 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
              >
                Browse
              </a>
              <a 
                href="#" 
                className="rounded-md px-3 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
              >
                Saved
              </a>
              <button className="ml-2 rounded-md bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 transition-colors">
                Sign In
              </button>
            </nav>
          </div>
        </div>
      </header>
      
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {children}
      </main>
      
      <footer className="mt-20 border-t border-neutral-200 bg-white">
        <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-8 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-600">
                <span className="text-xs font-bold text-white">P</span>
              </div>
              <p className="text-sm text-neutral-600">
                © {new Date().getFullYear()} PropertyList. All rights reserved.
              </p>
            </div>
            <div className="flex gap-6 text-sm">
              <a href="#" className="text-neutral-600 hover:text-neutral-900 transition-colors">
                About
              </a>
              <a href="#" className="text-neutral-600 hover:text-neutral-900 transition-colors">
                Contact
              </a>
              <a href="#" className="text-neutral-600 hover:text-neutral-900 transition-colors">
                Privacy
              </a>
              <a href="#" className="text-neutral-600 hover:text-neutral-900 transition-colors">
                Terms
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}