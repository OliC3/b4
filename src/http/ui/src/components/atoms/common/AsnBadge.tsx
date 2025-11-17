import React from "react";
import { B4Badge } from "./B4Badge";
import { useAsnLookup } from "@hooks/useDomainActions";

interface AsnBadgeProps {
  ip: string;
}

export const AsnBadge: React.FC<AsnBadgeProps> = React.memo(({ ip }) => {
  const asnName = useAsnLookup(ip);

  if (!asnName) return null;

  return <B4Badge badgeVariant="yellowOutline" label={asnName} />;
});

AsnBadge.displayName = "AsnBadge";
