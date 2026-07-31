import { PageHeader, type PageHeaderProps } from "@/components/layout/PageHeader";

/**
 * Kept as a thin alias so every existing page importing PageTheme picks up
 * the revamped header (brand icon tile, breadcrumbs, actions slot) with no
 * per-page migration. New pages should import PageHeader directly.
 */
const PageTheme = (props: PageHeaderProps) => <PageHeader {...props} />;

export default PageTheme;
