import os
import requests
import unittest
import sys

BASE_URL = os.getenv("API_URL", "http://localhost:8080")
ADMIN_USERNAME = os.getenv("ADMIN_USERNAME", "leonanf")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "rtccow52")

# Configura cores para terminal (opcional)
GREEN = '\033[92m'
YELLOW = '\033[93m'
RED = '\033[91m'
BLUE = '\033[94m'
RESET = '\033[0m'

def print_colored(text, color):
    if sys.platform == "win32":
        print(text)  # sem cor no Windows simples
    else:
        print(f"{color}{text}{RESET}")

class ImprovedTestResult(unittest.TextTestResult):
    def __init__(self, stream, descriptions, verbosity):
        super().__init__(stream, descriptions, verbosity)
        self.successes = []

    def addSuccess(self, test):
        super().addSuccess(test)
        self.successes.append(test)
        if self.showAll:
            self.stream.write(f" {GREEN}OK{RESET}\n")

    def addSkip(self, test, reason):
        super().addSkip(test, reason)
        if self.showAll:
            self.stream.write(f" {YELLOW}SKIPPED{RESET} ({reason})\n")

    def printSummary(self):
        total = self.testsRun
        passed = len(self.successes)
        skipped = len(self.skipped)
        failures = len(self.failures)
        errors = len(self.errors)
        print("\n" + "="*60)
        print(f"RESUMO DOS TESTES:")
        print(f"  Total: {total}")
        print_colored(f"  Passou: {passed}", GREEN if passed > 0 else RESET)
        if skipped > 0:
            print_colored(f"  Ignorado: {skipped}", YELLOW)
        if failures > 0:
            print_colored(f"  Falhou: {failures}", RED)
        if errors > 0:
            print_colored(f"  Erro: {errors}", RED)
        if skipped > 0:
            print("\nTestes ignorados:")
            for test, reason in self.skipped:
                print(f"  - {test.id()}: {reason}")
        print("="*60)

class TestCopasoftwareAPI(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.token = None
        cls.team_ids = []
        cls.participant_ids = []
        cls.admin_login()

    @classmethod
    def admin_login(cls):
        url = f"{BASE_URL}/admin/login"
        r = requests.post(url, json={"username": ADMIN_USERNAME, "password": ADMIN_PASSWORD})
        assert r.status_code == 200, f"Login falhou: {r.status_code}"
        cls.token = r.json()["data"]["token"]

    def admin_headers(self):
        return {"Authorization": f"Bearer {self.token}"}

    # ------------------------------------------------------------
    def test_01_add_team_names(self):
        """Adiciona nomes e verifica listagem."""
        names = ["Python", "Go", "Rust", "JavaScript", "Ruby"]
        for name in names:
            with self.subTest(name=name):
                r = requests.post(f"{BASE_URL}/admin/team-names",
                                  json={"name": name},
                                  headers=self.admin_headers())
                self.assertEqual(r.status_code, 201)

        r = requests.get(f"{BASE_URL}/admin/team-names/available", headers=self.admin_headers())
        self.assertEqual(r.status_code, 200)
        data = r.json()["data"]
        self.assertIsInstance(data, list)
        self.assertGreaterEqual(len(data), len(names))

    def test_02_add_duplicate_team_name(self):
        r = requests.post(f"{BASE_URL}/admin/team-names",
                          json={"name": "Python"},
                          headers=self.admin_headers())
        self.assertEqual(r.status_code, 409)

    # ------------------------------------------------------------
    def test_03_signup_individual_success(self):
        participants = [
            ("2023001", "João Silva", 3),
            ("2023002", "Maria Souza", 5),
            ("2023003", "Pedro Santos", 2),
        ]
        for mat, nome, sem in participants:
            with self.subTest(matricula=mat):
                r = requests.post(f"{BASE_URL}/signup/individual", json={
                    "matricula": mat, "nome": nome, "semestre": sem
                })
                self.assertEqual(r.status_code, 201)
                data = r.json()["data"]
                self.assertIn("id", data)
                self.participant_ids.append(data["id"])

    def test_04_signup_individual_duplicate(self):
        r = requests.post(f"{BASE_URL}/signup/individual", json={
            "matricula": "2023001", "nome": "Outro", "semestre": 4
        })
        self.assertEqual(r.status_code, 409)

    def test_05_signup_individual_invalid_semester(self):
        r = requests.post(f"{BASE_URL}/signup/individual", json={
            "matricula": "2023999", "nome": "Invalido", "semestre": 0
        })
        self.assertEqual(r.status_code, 400)

    # ------------------------------------------------------------
    def test_06_signup_team_success(self):
        participants = [
            {"matricula": "2023004", "nome": "Ana Lima", "semestre": 4},
            {"matricula": "2023005", "nome": "Lucas Rocha", "semestre": 1},
            {"matricula": "2023006", "nome": "Carla Mendes", "semestre": 3}
        ]
        r = requests.post(f"{BASE_URL}/signup/team", json={"participants": participants})
        self.assertEqual(r.status_code, 201)
        data = r.json()["data"]
        self.assertIn("id", data)
        self.assertEqual(data["status"], "pending")
        self.assertEqual(len(data["participantData"]), 3)
        self.assertIsNone(data.get("participants"))
        self.team_ids.append(data["id"])

    def test_07_signup_team_with_existing_matricula(self):
        participants = [
            {"matricula": "2023001", "nome": "João Silva", "semestre": 3},
            {"matricula": "2023007", "nome": "Novo", "semestre": 2},
            {"matricula": "2023008", "nome": "Outro", "semestre": 4}
        ]
        r = requests.post(f"{BASE_URL}/signup/team", json={"participants": participants})
        self.assertEqual(r.status_code, 409)

    def test_08_signup_team_insufficient_participants(self):
        participants = [{"matricula": "2023009", "nome": "Um", "semestre": 2}]
        r = requests.post(f"{BASE_URL}/signup/team", json={"participants": participants})
        self.assertEqual(r.status_code, 400)

    # ------------------------------------------------------------
    def test_09_approve_team(self):
        if not self.team_ids:
            self.skipTest("Nenhum time pendente")
        team_id = self.team_ids[0]
        r = requests.post(f"{BASE_URL}/admin/teams/{team_id}/approve",
                          headers=self.admin_headers())
        self.assertEqual(r.status_code, 200)

        r2 = requests.get(f"{BASE_URL}/teams/{team_id}")
        self.assertEqual(r2.status_code, 200)
        team = r2.json()["data"]
        self.assertEqual(team["status"], "approved")
        self.assertIn("code", team)
        self.assertIn("name", team)
        self.assertIsInstance(team.get("participants"), list)
        self.assertEqual(len(team["participants"]), 3)

    def test_10_approve_already_approved(self):
        if not self.team_ids:
            self.skipTest("Nenhum time")
        r = requests.post(f"{BASE_URL}/admin/teams/{self.team_ids[0]}/approve",
                          headers=self.admin_headers())
        self.assertIn(r.status_code, [400, 409])

    # ------------------------------------------------------------
    def test_11_reject_team(self):
        """Cria um time pendente e o rejeita, verificando liberação do nome."""
        # 1. Criar time pendente
        participants = [
            {"matricula": "2023010", "nome": "Rejeitado1", "semestre": 2},
            {"matricula": "2023011", "nome": "Rejeitado2", "semestre": 4},
            {"matricula": "2023012", "nome": "Rejeitado3", "semestre": 5}
        ]
        r = requests.post(f"{BASE_URL}/signup/team", json={"participants": participants})
        self.assertEqual(r.status_code, 201)
        team_data = r.json()["data"]
        team_id = team_data["id"]
        team_name = team_data["name"]  # nome reservado

        # 2. Verificar nomes disponíveis antes da rejeição
        r_avail_before = requests.get(f"{BASE_URL}/admin/team-names/available",
                                    headers=self.admin_headers())
        self.assertEqual(r_avail_before.status_code, 200)
        avail_before = len(r_avail_before.json()["data"])

        # 3. Rejeitar o time
        r_reject = requests.post(f"{BASE_URL}/admin/teams/{team_id}/reject",
                                headers=self.admin_headers())
        self.assertEqual(r_reject.status_code, 200)

        # 4. Verificar status do time
        r_team = requests.get(f"{BASE_URL}/teams/{team_id}")
        self.assertEqual(r_team.status_code, 200)
        self.assertEqual(r_team.json()["data"]["status"], "rejected")

        # 5. Verificar se o nome foi liberado
        r_avail_after = requests.get(f"{BASE_URL}/admin/team-names/available",
                                    headers=self.admin_headers())
        self.assertEqual(r_avail_after.status_code, 200)
        avail_after = len(r_avail_after.json()["data"])
        self.assertEqual(avail_after, avail_before + 1, "Nome não foi liberado")

    # ------------------------------------------------------------
    def test_13_public_list_teams(self):
        r = requests.get(f"{BASE_URL}/teams")
        self.assertEqual(r.status_code, 200)
        self.assertIsInstance(r.json()["data"], list)

    def test_14_public_list_participants(self):
        r = requests.get(f"{BASE_URL}/participants")
        self.assertEqual(r.status_code, 200)
        self.assertIsInstance(r.json()["data"], list)

    def test_15_public_get_participant_by_matricula(self):
        r = requests.get(f"{BASE_URL}/participants/matricula/2023001")
        self.assertEqual(r.status_code, 200)
        self.assertEqual(r.json()["data"]["matricula"], "2023001")

    # ------------------------------------------------------------
    def test_16_signup_individual_after_team_approval(self):
        r = requests.post(f"{BASE_URL}/signup/individual", json={
            "matricula": "2023004", "nome": "Tentativa", "semestre": 2
        })
        self.assertEqual(r.status_code, 409)

    def test_17_create_team_with_matricula_from_approved_team(self):
        participants = [
            {"matricula": "2023004", "nome": "Repetido", "semestre": 3},
            {"matricula": "2023013", "nome": "Novo1", "semestre": 2},
            {"matricula": "2023014", "nome": "Novo2", "semestre": 4}
        ]
        r = requests.post(f"{BASE_URL}/signup/team", json={"participants": participants})
        self.assertEqual(r.status_code, 409)

if __name__ == "__main__":
    # Configurar runner com resultado customizado
    suite = unittest.TestLoader().loadTestsFromTestCase(TestCopasoftwareAPI)
    runner = unittest.TextTestRunner(resultclass=ImprovedTestResult, verbosity=2)
    result = runner.run(suite)
    result.printSummary()