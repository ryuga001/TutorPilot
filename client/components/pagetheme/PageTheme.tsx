import React from 'react'

interface PageThemeProps {
  title?: string;
  subtitle?: string;
  children?: React.ReactNode;
}

const PageTheme = ({ title, subtitle, children }: PageThemeProps) => {
  return (
    <div className="space-y-4 p-2 lg:px-8">
      <section className="space-y-2">
        {title && (
          <h1 className="text-3xl font-bold tracking-tight">
            {title}
          </h1>
        )}

        {subtitle && (
          <p className="max-w-3xl text-sm text-muted-foreground">
            {subtitle}
          </p>
        )}
      </section>

      <section className="space-y-6">
        {children}
      </section>
    </div>
  );
};

export default PageTheme;