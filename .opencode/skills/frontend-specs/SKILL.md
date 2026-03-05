---
name: frontend-specs
description: Rules for creating and editing frontend components with React, oat and scss. Use when making any frontend changes
---

The frontend is located in the `frontend/src` directory. Follow these guidelines when creating or editing React
components:

## Tech Stack

- **React**: 19.1.0
- **Oat UI**: 0.3.0 - [Components](https://oat.ink/components/)
- **Icons**: lucide-react
- **Styling**: SASS (SCSS)

## Project Structure

- `src/pages/`: Contains full-page components. Each page should have its own directory containing:
    - components are named with kebab-case (e.g., `page-name.tsx`)
    - `page-name.tsx`: The main component.
    - `page-name.scss`: Component-specific styles.
- `src/shared/`: Contains shared resources. Additional folders can be created
    - `shared/components/`: Reusable UI components.
    - `shared/styles/`: Global stylesheets and shared SASS variables/mixins.
    - Use HashRouter for routing.
    - Always show a list of files changed.
    - Imports should always use tsconfig path alias

## Styling & Theme

- **Oat UI**: Oat provides CSS-based styling for semantic HTML elements. Use oat's CSS classes and data attributes for components.
- **Theme**: Use `data-theme="dark"` attribute on `<html>` element for dark mode. Oat uses CSS variables for theming.
- **Style Guidelines**:
    - Each component should have its own `.scss` file.
    - Use class names that match the component name (e.g., `dashboard.scss` for `dashboard.tsx`).
    - Prefer using semantic HTML elements with oat classes over custom components.
    - Do not use inline styles unless absolutely necessary (use CSS variables instead).
    - Use oat CSS variables for values already provided by oat (found in frontend/node_modules/@knadh/oat/css/01-theme.css).
    - Very Important When writing SCSS attributes, always use parent selector(&) to build on new selectors example `.settings { &-container {}}`

## Component Patterns

- Use functional components with `React.FC` or `FC`.
- Use semantic HTML elements with oat classes (e.g., `<button data-variant="primary">`, `<table class="table">`).
- Use lucide-react for icons (e.g., `import { Search, Bell } from 'lucide-react';`).
- Always import styles directly in the component file (e.g., `import './page-name.scss';`).

