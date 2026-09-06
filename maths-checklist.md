# Maths checklist — sirf itna, jitna is project mein chahiye

**Update:** hum **local plane** pe kaam kar rahe hain (x, y metres — degrees nahi).
Isse maths bahut kam ho gaya. Haversine, arc length, cos(latitude) — sab hat gaye.

**Loop 0 ke liye: teen cheezein. ~1 ghanta.**

---

## Kaise use karna hai

Har item ke saath **English search term** hai (search English mein karna) aur ek **check sawal**.

Check ka jawab bina dekhe aa gaya → ho gaya, aage badho.

Agar koi video/page in sawalon se aage ki cheezein sikhane lage — **band kar do.** Wo tumhare kaam ka nahi hai.

---

## Loop 0 — teen cheezein

### 1. Pythagoras theorem

`Pythagorean theorem` · `distance between two points formula`

```
a² + b² = c²
```

Aur usi se, do points ke beech doori:

```
doori = √( (x2−x1)² + (y2−y1)² )
```

**Kyun chahiye:** cow aur fence ke kone ke beech doori. Cow kitni door chali.

> **Check:** ek point `(0,0)` pe, doosra `(3,4)` pe. Doori kitni?

---

### 2. sin aur cos — right triangle ✓ *(cos ho gaya)*

`sine cosine right triangle` · `SOHCAHTOA`

```
cos(φ) = adjacent ÷ hypotenuse      ← ho gaya
sin(φ) = opposite  ÷ hypotenuse      ← bacha hai
```

**Kyun chahiye:** cow ek direction (heading) mein chalti hai. Us heading ko x aur y mein todne ke liye:

```
dx = distance × sin(heading)
dy = distance × cos(heading)
```

> **Check:** `cos(60°)` kitna hai? `sin(0°)` kitna?
> Kya `sin` ya `cos` kabhi 1 se bada ho sakta hai?

---

### 3. Radians ✓ *(ho gaya)*

`radians vs degrees` · `convert degrees to radians`

```
360° = 2π radians       degrees × π/180 = radians
```

**Kyun chahiye:** Go ka `math.Sin` aur `math.Cos` **radians** lete hain, degrees nahi.
Bhool gaye to answer galat aayega aur koi error nahi dikhega.

> **Check:** 90° radians mein kitna hai?

---

## Optional — chaho to abhi, warna baad mein

### 4. Normal distribution aur standard deviation

`normal distribution explained` · `standard deviation intuition` · `68 95 99.7 rule`

Bell curve, aur **σ (sigma)** ka matlab.

**Kyun:** cow ka turn angle isse zyada asli lagta hai (chhote mud zyada, tez mud kam).
Aur Loop 2 mein GPS noise (σ ≈ 3–5 m).

**Skip kar sakte ho** — abhi simple random turn se kaam chal jaayega. "Basic cow pehle."

> **Check:** σ = 3 metre. Lagbhag kitne percent readings 3 metre ke andar aayengi?

---

## Baad ke liye — jab GPS aayega (Loop 2)

### 5. cos(latitude) — degrees se metres

Ye tum **samajh chuke ho.** Aur ab ye poore system mein sirf **do line** hai,
ek boundary file mein:

```
x = (lng − originLng) × 111195 × cos(originLat)
y = (lat − originLat) × 111195
```

Kuch naya nahi seekhna. Bas jab wo waqt aaye, ye do line likhni hain.

---

## ⛔ Bilkul nahi chahiye — ek minute mat lagana

Trigonometry search karte hi ye sab saamne aayega. **Band kar dena.**

- ~~Haversine~~ — local plane pe zaroorat hi nahi
- ~~Arc length, circumference, radians ka geometric matlab~~
- ~~atan2~~ — abhi nahi
- ~~Koi bhi proof ya derivation~~
- ~~Trig identities (sum, double angle, product)~~
- ~~Unit circle ka poora chapter~~
- ~~Sine rule, cosine rule, triangle solving~~
- ~~Calculus — derivative, integral, limits~~
- ~~Map projections — Mercator, UTM~~
- ~~Geodesy, ellipsoid, Vincenty~~
- ~~Spherical trigonometry~~
- ~~Linear algebra, matrices, vectors~~

---

## Yaad rakhna

Loop 3 se 10 tak maths ki **ek line nahi** hai.
Wahan MQTT hai, durability hai, Kubernetes hai, Terraform hai —
**aur wahi is project ka sabse bada hissa hai.**

Maths buniyad hai. Buniyad poori imaarat nahi hoti.
