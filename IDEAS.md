# IDEAS

Feature ideas jo abhi ke DoD ka hissa nahi hain.

**Niyam:** idea aaye → yahan likho → kaam pe wapas jao.
Banao sirf tab jab wo *current* loop ke Definition of Done mein ho.

---

## Loop 1

- **Cue response.** `Cow` mein `Heading` field + `TurnAway()` method.
  Collar cue bajne pe bulaye. Cow probability *p* se mude
  (trained ≈ 0.85, pulse pe ≈ 0.98, untrained cows kam).
  Isse "cue compliance rate" metric nikalta hai.
  *(sin/cos chahiye — abhi nahi banaya)*

- **Correlated random walk.** Abhi cow jitter karti hai. Heading ko
  yaad rakho aur har step thoda ghumao, nayi random direction mat lo.

- **Herd cohesion.** Jhund ke centre ki taraf halka kheenchav.
  Iske bina Loop 5 ka "herd se alag gaay" detector bekaar hai.

- **Activity modes.** grazing / walking / resting — alag speed aur
  alag turn rate, realistic dwell time ke saath.

---

## Loop 2

- **GPS noise.** Collar ke andar — asli position padho, uspe error daalo.
  σ ≈ 3–5 m, kabhi-kabhi 20 m ka outlier.

- **Hysteresis + dwell time.** Andar ghusne aur bahar nikalne ki
  alag-alag lines. State badalne se pehle N second ka intezaar.

---

## Baad ke liye

- **Escalation ki aakhri seedhi:** N koshishon ke baad collar haar maan le,
  cue band kare, aur farmer ko flag kare.
- **Breach episode id** — ek escape = ek episode = ek alert.
