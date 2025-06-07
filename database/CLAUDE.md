# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in database, which contains the database schema for our application.

## Database Schema
Multi-tenant structure with:
- `tenants`: Organization accounts with plan tiers
- `users`: User accounts with roles and statuses
- `refresh_tokens`: JWT refresh token storage
- `password_reset_tokens`: Password reset flow
- `invitation_tokens`: User invitation system
- `email_change_tokens`: Email change verification
