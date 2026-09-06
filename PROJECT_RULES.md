# Seyra Project Rules

## Project Overview

Project:
Seyra

Type:
Secure encrypted messaging application

Platforms:
- iOS
- Android

Goal:
Build a secure communication platform similar to Telegram and WhatsApp with strong privacy, security, and encryption.

The application must prioritize:
- Security
- Privacy
- Performance
- Scalability
- Clean architecture
- User experience


# Product Features

## Communication

- End-to-end encrypted private messages
- Encrypted voice calls
- Encrypted video calls
- Real-time messaging
- Message delivery status
- Message history
- Message deletion
- Media messages


## Groups

- Private groups
- Public groups
- Group administrators
- Permissions system
- Member management
- Group security controls


## Channels

- Public channels
- Private channels
- Channel subscriptions
- Channel administrators
- Broadcasting messages


## User System

- Username-based accounts
- Password authentication
- User profiles
- Profile pictures
- Privacy settings
- User blocking
- Account management


## File Sharing

- Secure file uploads
- Images
- Videos
- Documents
- Audio files
- Encrypted file storage


## Bots

- Bot accounts
- Bot API system
- Automation features
- Secure permissions


## Premium Features

- Subscription system
- Premium accounts
- Additional storage
- Advanced features


## Account Management

- Delete account permanently button
- Remove all user data
- Secure account termination process


# Technical Architecture

## Frontend

Technology:
- Flutter
- Dart

Platforms:
- Android
- iOS


## Backend

Requirements:
- Secure backend architecture
- Scalable infrastructure
- Real-time communication support
- Strong authentication system
- Secure database design


## Encryption

Requirements:

- End-to-end encryption for private communication
- Secure key management
- Encrypted message transmission
- Encrypted file handling
- Secure storage of encryption keys


# Security Requirements

Important:
Security is the highest priority.

Rules:

- Never store passwords in plain text.
- Use secure password hashing.
- Use secure storage for tokens and keys.
- Minimize collected user data.
- Protect user privacy.
- Validate all user input.
- Prevent unauthorized access.
- Follow secure coding practices.
- Never expose private keys.
- Never weaken encryption for convenience.


# Development Strategy

Build Seyra in phases.


## Phase 0: Foundation

Current tasks:

- Rename project to Seyra
- Configure Flutter environment
- Configure Android package
- Configure iOS package
- Setup project structure
- Setup clean architecture


## Phase 1: Application Foundation

Build:

- Clean architecture
- Folder structure
- App theme
- Design system
- Navigation system
- Authentication foundation
- User profile system


## Phase 2: Messaging Core

Build:

- Real-time messaging
- Chat interface
- Message storage
- Encryption layer
- Message synchronization


## Phase 3: Communication Features

Build:

- Voice calls
- Video calls
- Groups
- Channels
- File sharing


## Phase 4: Advanced Features

Build:

- Premium features
- Bots
- Advanced privacy settings
- Production optimization
- App store preparation


# Flutter Development Rules

- Use clean architecture.
- Keep features modular.
- Separate UI, business logic, and data layers.
- Avoid unnecessary dependencies.
- Prefer stable and maintained packages.
- Write readable production-quality code.
- Keep code scalable.
- Do not create unnecessary files.
- Follow Flutter best practices.


# Project Structure Rules

Preferred structure:

lib/

    core/
        constants/
        errors/
        network/
        security/
        storage/
        theme/

    features/

        auth/
            data/
            domain/
            presentation/

        chat/
            data/
            domain/
            presentation/

        calls/
            data/
            domain/
            presentation/

        groups/

        channels/

        profile/

    shared/

    main.dart


# AI Assistant Rules

When working on Seyra:

- Read this file before making changes.
- Understand the architecture before coding.
- Explain major technical decisions.
- Ask before changing architecture.
- Do not rewrite working code without reason.
- Do not add dependencies without explaining why.
- Keep security as the priority.
- Avoid shortcuts that reduce security.
- Build step-by-step.
- Complete one feature before moving to another.
- Prefer long-term maintainability over quick solutions.


# Cursor Agent Instructions

Before coding:

1. Analyze the current project.
2. Explain the implementation plan.
3. Wait for confirmation before major changes.

When coding:

- Modify only necessary files.
- Explain what changed.
- Mention possible risks.
- Keep the project compiling after changes.


# Current Status

Completed:

- Flutter project created
- Android environment configured
- Android emulator working
- Project renamed to Seyra


Current Goal:

Create the professional foundation of Seyra before implementing features.


Next Tasks:

1. Create clean architecture structure.
2. Setup application theme.
3. Setup routing/navigation.
4. Setup dependency injection.
5. Prepare authentication foundation.


# Final Principle

Seyra is a security-focused communication platform.

Every technical decision must consider:

Security first.
Privacy first.
Scalability first.
User experience always.