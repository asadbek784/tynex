# Tynex CLI — DOIMIY QOIDALAR (hech qachon o'tkazib yuborma)

═══════════════════════════════════════════════════════════════
BU QOIDALAR knowledge.md faylida saqlangan. Kelajakda men bu
qoidalarni qayta yozmasligim kerak, Buffy ularni har bir
vazifada o'zi eslab, qo'llaydi.
═══════════════════════════════════════════════════════════════


## 8. MAJBURIY ISHLASH KETMA-KETLIGI (HAR BIR O'ZGARISHDAN KEYIN)

Bu tartibni hech qachon o'tkazib yuborma, qisqartirma:

1. `go build ./...` → xato bo'lsa tuzat, qayta tekshir
2. `go vet ./...`
3. `git status` → natijani ko'rsat
4. Maxfiy fayl tekshiruvi (config.yaml/.env/*.key commit'da YO'Q)
5. `git add .`
6. `git commit -m "<aniq, mazmunli xabar>"`
7. `git remote -v` → bo'lmasa, token orqali qo'sh
8. `git push -u origin main`
9. Push natijasini tekshir


## 9. YAKUNIY XULOSADA HAR DOIM 3 NARSA ALOHIDA KO'RSATILSIN

- Build holati: ✅ muvaffaqiyatli / ❌ xato (sababi)
- Git holati: ✅ commit+push qilindi / ❌ qilinmadi (sababi)
- GitHub havolasi: https://github.com/<username>/tynex

Bu 3 qatorsiz "tayyor" deb hech qachon tugatma.


## 10. XATO YUZAGA KELGANDA

- To'liq xato matnini ko'rsat, sababini tushuntir
- Tasdiqsiz qo'shimcha amal qilma, ruxsat so'ra


## 11. ARXITEKTURA PRINSIPI (O'ZGARMAS)

- **Provider-agnostic** printsipiga hech qachon zid kod yozilmasin
- Eski/ishlatilmaydigan kod (bitta provayderga qattiq bog'langan)
  darhol olib tashlansin


## 12. YANGI VAZIFA QABUL QILGANDA

- Avval mavjud kodni o'qib, qaysi fayllarga ta'sir qilishini aniqla
- Keyin o'zgartir, keyin 8-bo'limdagi ketma-ketlikni to'liq bajar


## 13. BU QOIDALARNING KELIB CHIQISHI

- knowledge.md fayli birinchi bosqichda yaratilgan va saqlanadi
- Har bir vazifada Buffy bu qoidalarni o'zi eslab, qo'llaydi
- Qayta yozish shart emas, doimiy ravishda amal qilinadi


## QO'SHIMCHA QOIDALAR

### Konventsiyalar:
- Mavjud kod konventsiyalariga qat'iy rioya qil
- Yangi kutubxona/ramkalarni taxmin qilma — imports va config'larni tekshir
- Kod uslubi, tuzilishi, tiplari bo'yicha mavjud kodga moslash

### O'zgartirish uslubi:
- Minimal o'zgartirish — faqat so'ralgan narsani qil
- Har bir kod qatorining maqsadi bor deb hisobla
- Mavjud helper/komponentlarni qayta ishlat

### Gigiyena:
- Importlarni unutma
- Ishlatilmaydigan o'zgaruvchi/funksiya/fayllarni olib tashla
- "any" tipiga cast qilma (haqiqatan har qanday tip bo'lishi mumkin bo'lgan hollar bundan mustasno)

### Testing:
- Unit test yaratilsa, albatta ishga tushir va o'tkaz

### Paket boshqaruvi:
- Yangi paket qo'shganda `basher` orqali o'rnat, paket versiyasini taxmin qilma
- Global paket o'rnatma (so'ralmasa)
- Loyihaning paket menejerini ishlat (npm, pnpm, yarn va h.k.)

### Frontend:
- UI ni imkon qadar chiroyli qil
- Hover, transition, mikro-interaksiyalarni qo'sh
- Dizayn prinsiplarini qo'lla (ierarxiya, kontrast, balans)

### Refactoring:
- Eksport qilinadigan simvol (funksiya/klass/o'zgaruvchi) o'zgartirilsa,
  barcha reference'larni top va yangila (code-searcher orqali)


## BUYRUQ STRUKTURASI

```
tynex config add/list/set/delete <path>
tynex use <name>
tynex chat / tynex
tynex -p "prompt" / tynex <prompt>
tynex init      — boshlang'ich sozlash
tynex session   — sessiyalarni boshqarish
```

## XAVFSIZLIK

- Yozish/o'chirish/shell'dan oldin HAR DOIM tasdiq so'rash
- API kalit hech qachon log/kod/commit'ga yozilmasin
- .gitignore: *.env, config.yaml, .tynex/, *.key, *.pem, *.log
