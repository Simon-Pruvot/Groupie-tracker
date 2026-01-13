# Groupie Tracker

## Description
**Groupie Tracker** est une application web écrite en Go qui permet de consulter des informations sur différents artistes et groupes de musique. L'application consomme une API externe pour récupérer et afficher des données telles que les membres du groupe, les dates de concerts, les lieux, les dates de création, et les premiers albums.

Ce projet a été réalisé dans le cadre d'un exercice de programmation.

## Fonctionnalités
- **Liste des artistes** : Affichage de tous les artistes disponibles via l'API.
- **Détails de l'artiste** : Page dédiée pour chaque artiste affichant :
  - Image et nom
  - Liste des membres
  - Date de création
  - Premier album
  - Genres musicaux
  - Concerts (Dates et Lieux)
- **Recherche** : Barre de recherche pour trouver un artiste par son nom.
- **Pages statiques** :
  - Home 
  - Contact

## Technologies utilisées
- **Backend** : Go (Golang)
- **Frontend** : Templates HTML (`html/template`) et CSS
- **API** : [Groupie Trackers API](https://groupietrackers.herokuapp.com/api)

## Structure du projet
```text
/
├── server.exe          # Exécutable (si compilé)
├── src/
│   ├── main.go         # Point d'entrée principal de l'application
│   ├── CSS/            # Feuilles de style (index.css, style.css, etc.)
│   ├── templates/      # Fichiers HTML (index.html, header.html, etc.)
│   ├── images/         # Images statiques
│   └── musique/        # Fichiers audio
```

## Installation et Lancement

### Prérequis
- Avoir [Go](https://go.dev/dl/) installé sur votre machine.

### Cloner le projet
```bash
git clone https://github.com/votre-user/Groupie-tracker.git
cd Groupie-tracker
```

### Lancer l'application
Vous pouvez lancer le serveur directement avec la commande `go run` :

```bash
go run src/main.go
```

Une fois le serveur démarré, ouvrez votre navigateur et accédez à :
[http://localhost:8080](http://localhost:8080)

## API Endpoints utilisés
L'application interagit avec les endpoints suivants :
- `/artists` : Informations générales sur les artistes.
- `/locations` : Lieux des concerts.
- `/dates` : Dates des concerts.
- `/relation` : Relation entre les dates et les lieux.

## Auteur
Ce projet a été développé par Simon Pruvot et Romain.
