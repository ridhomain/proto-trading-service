local claims = {
  email_verified: false,
} + std.extVar('claims');

{
  identity: {
    traits: {
      // The email - only include if verified
      [if 'email' in claims && claims.email_verified then 'email' else null]: claims.email,
      
      // Name object with first and last name
      name: {
        [if 'given_name' in claims then 'first' else null]: claims.given_name,
        [if 'family_name' in claims then 'last' else null]: claims.family_name,
      },
      
      // Set default role as trader
      role: 'trader',
      
      // Store Google ID for reference
      [if 'sub' in claims then 'google_id' else null]: claims.sub,
      
      // Profile picture from Google
      [if 'picture' in claims then 'avatar_url' else null]: claims.picture,
    },
  },
}