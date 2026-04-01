# security_groups.tf

# ── ALB security group ───────────────────────────────────
resource "aws_security_group" "alb" {
  name        = "${var.project}-alb-sg"
  description = "ALB - allows HTTPS/HTTP from internet"
  vpc_id      = aws_vpc.main.id

  tags = {
    Name      = "${var.project}-alb-sg"
    Project   = var.project
    ManagedBy = "terraform"
  }
}

# ── Workers security group ───────────────────────────────
resource "aws_security_group" "workers" {
  name        = "${var.project}-workers-sg"
  description = "Worker nodes - accepts traffic from ALB only"
  vpc_id      = aws_vpc.main.id

  tags = {
    Name      = "${var.project}-workers-sg"
    Project   = var.project
    ManagedBy = "terraform"
  }
}

# ── DB security group ────────────────────────────────────
resource "aws_security_group" "db" {
  name        = "${var.project}-db-sg"
  description = "Database - accepts connections from workers only"
  vpc_id      = aws_vpc.main.id

  tags = {
    Name      = "${var.project}-db-sg"
    Project   = var.project
    ManagedBy = "terraform"
  }
}

# ── ALB rules ────────────────────────────────────────────
resource "aws_vpc_security_group_ingress_rule" "alb_https" {
  security_group_id = aws_security_group.alb.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 443
  to_port           = 443
  ip_protocol       = "tcp"
  description       = "HTTPS from internet"
}

resource "aws_vpc_security_group_ingress_rule" "alb_http" {
  security_group_id = aws_security_group.alb.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 80
  to_port           = 80
  ip_protocol       = "tcp"
  description       = "HTTP from internet - redirect to HTTPS"
}

resource "aws_vpc_security_group_egress_rule" "alb_all_out" {
  security_group_id = aws_security_group.alb.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
  description       = "All outbound"
}

# ── Workers rules ────────────────────────────────────────
resource "aws_vpc_security_group_ingress_rule" "workers_from_alb" {
  security_group_id            = aws_security_group.workers.id
  referenced_security_group_id = aws_security_group.alb.id
  from_port                    = 8080
  to_port                      = 8080
  ip_protocol                  = "tcp"
  description                  = "App traffic from ALB only"
}

resource "aws_vpc_security_group_egress_rule" "workers_all_out" {
  security_group_id = aws_security_group.workers.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
  description       = "All outbound - ECR pulls, API calls"
}

# ── DB rules ─────────────────────────────────────────────
resource "aws_vpc_security_group_ingress_rule" "db_from_workers" {
  security_group_id            = aws_security_group.db.id
  referenced_security_group_id = aws_security_group.workers.id
  from_port                    = 5432
  to_port                      = 5432
  ip_protocol                  = "tcp"
  description                  = "PostgreSQL from workers only"
}

resource "aws_vpc_security_group_egress_rule" "db_all_out" {
  security_group_id = aws_security_group.db.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
  description       = "All outbound - irrelevant, no route exists"
}