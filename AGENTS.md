# Agent Guidelines for pg-cluster

## Build/Test Commands
- `pnpm dev` - Run development server with pretty logging
- `pnpm install` - Install dependencies
- `tsc --noEmit` - Type check without emitting files
- No test framework configured yet

## Code Style Guidelines
- **Language**: TypeScript with ES2022 target, ESM modules
- **Imports**: Use `.ts` extensions, namespace imports for large libraries (`import * as k8s`)
- **Formatting**: Strict TypeScript config with `noImplicitOverride`
- **Types**: Explicit typing preferred, use `@types/node` for Node.js APIs
- **Naming**: camelCase for variables/functions, PascalCase for classes
- **Error Handling**: Use structured logging with pino logger
- **Dependencies**: Kubernetes client-node for K8s operations, pino for logging

## Project Structure
- `src/` - Main source code (TypeScript only)
- `external-repo/` - Reference repos (kubernetes client, k8s-operator) - READ ONLY
- Entry point: `src/index.ts`

## Important Notes
- Never modify files in `external-repo/` - use only as reference
- Use pino logger from `src/logger.ts` for all logging
- Follow existing import patterns with namespace imports for large libraries