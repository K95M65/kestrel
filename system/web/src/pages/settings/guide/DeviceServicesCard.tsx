import { LIFE_RECIPES } from "@/lib/lifeRecipes";
import { GuideConnectStep } from "@/pages/settings/guide/GuideConnectStep";

/** Device → Channels: same connect forms as Guided Setup, all home-user services. */
export function DeviceServicesCard() {
  return (
    <div style={{ marginTop: 22, paddingTop: 18, borderTop: "1px solid var(--lm-border)" }}>
      <div style={{ fontSize: 13, fontWeight: 650, marginBottom: 4 }}>Mail and calendar</div>
      <GuideConnectStep
        lead="Gmail uses an app password. Calendar uses the secret iCal address from Google Calendar. Mail stays a draft."
        services={LIFE_RECIPES.desk.services.filter((s) => s.kind === "connector")}
      />
    </div>
  );
}
