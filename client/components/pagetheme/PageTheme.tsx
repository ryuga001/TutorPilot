import React from 'react'

interface PageThemeProps {
    title ?: string;
    subtitle ?: string;
    children ?: React.ReactNode;
}

const PageTheme = ({ title, subtitle, children }: PageThemeProps) => {
  return (
    <div>
        <section>
            <div>
                {title && <h1 className='text-2xl font-bold'>{title}</h1>}
            </div>
            <div>
                {subtitle && <p className='text-sm text-muted-foreground'>{subtitle}</p>}
            </div>
        </section>
        <section>
            {children}
        </section>
    </div>
  )
}

export default PageTheme