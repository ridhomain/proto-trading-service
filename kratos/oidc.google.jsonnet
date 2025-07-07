local claims = {
  email_verified: false,
} + std.extVar('claims');

{
  identity: {
    traits: {
      // The email might be empty if the user hasn't granted permissions for the email scope.
      [if "email" in claims && claims.email_verified then "email" else null]: claims.email,
      
      // Google provides name in different formats
      name: {
        first: if "given_name" in claims then claims.given_name else "User",
        last: if "family_name" in claims then claims.family_name else "",
      },
      
      // Set default role as trader
      role: "trader",
      
      // Store Google ID for reference
      google_id: claims.sub,
      
      // Profile picture from Google
      [if "picture" in claims then "avatar_url" else null]: claims.picture,
    },
  },
}