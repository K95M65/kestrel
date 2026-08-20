import { LiveFaceEnroll } from "@/pages/settings/guide/LiveFaceEnroll";

export function GuideSeeStep({
  onTried, onEnrolled, robotName, lead,
}: {
  onTried: () => void;
  onEnrolled?: (name: string) => void;
  robotName?: string;
  lead?: string;
}) {
  const who = robotName?.trim() || "the robot";
  return (
    <LiveFaceEnroll
      robotName={robotName}
      speak
      onTried={onTried}
      onEnrolled={onEnrolled}
      lead={lead || `We'll add you as the first person. Stand in front of ${who}, take a photo, then say who you are.`}
    />
  );
}
