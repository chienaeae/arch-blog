# Supabase Configuration Guide

This guide outlines the final steps required to connect this repository to your Supabase project and enable the full database branch workflow.

These steps must be performed manually as they require access to your Supabase project dashboard and your repository's CI/CD settings.

## 1. Link Your Supabase Project

To enable the Supabase CLI to work with your remote project (e.g., for database branching), you need to link this local repository to your Supabase project.

Run the following command and follow the prompts. You will need your project's reference ID, which can be found in your Supabase project's URL (`https://app.supabase.com/project/<PROJECT_REF>`).

```bash
npx supabase link --project-ref <YOUR_PROJECT_REF>
```

## 2. Configure CI/CD Secrets

For the CI/CD pipeline to interact with Supabase (e.g., to create database branches for pull requests), you must add the following secrets to your repository's CI/CD configuration (e.g., GitHub Actions secrets):

- `SUPABASE_ACCESS_TOKEN`: Your Supabase personal access token. You can generate one from your [Supabase account page](https://app.supabase.com/account/tokens).
- `SUPABASE_PROJECT_REF`: The reference ID of your Supabase project.
- `DATABASE_URL`: The connection string for your database. You will likely need separate ones for different environments (e.g., staging, production). Ensure you create a dedicated, least-privilege user/role for the CI seeder, as specified in the AIP.

Once these steps are complete, the Supabase integration for Phase 1 should be fully configured.
