# igw_nat.tf

# ── Internet Gateway ─────────────────────────────────────
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name      = "${var.project}-igw"
    Project   = var.project
    ManagedBy = "terraform"
  }
}

# ── Elastic IPs for NAT Gateways (one per AZ) ────────────
resource "aws_eip" "nat" {
  for_each = toset(var.azs)

  domain = "vpc"

  tags = {
    Name      = "${var.project}-nat-eip-${each.key}"
    Project   = var.project
    ManagedBy = "terraform"
  }

  depends_on = [aws_internet_gateway.main]
}

# ── NAT Gateways (one per AZ, in public subnets) ─────────
resource "aws_nat_gateway" "main" {
  for_each = toset(var.azs)

  allocation_id = aws_eip.nat[each.key].id
  subnet_id     = aws_subnet.public[
    "public-${index(var.azs, each.key) + 1}${substr(each.key, -1, 1)}"
  ].id

  tags = {
    Name      = "${var.project}-nat-${each.key}"
    Project   = var.project
    ManagedBy = "terraform"
  }

  depends_on = [aws_internet_gateway.main]
}

# ── Public route table (one, shared by all public subnets)
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name      = "${var.project}-rt-public"
    Project   = var.project
    ManagedBy = "terraform"
  }
}

# ── Associate all 3 public subnets → public route table ──
resource "aws_route_table_association" "public" {
  for_each = aws_subnet.public

  subnet_id      = each.value.id
  route_table_id = aws_route_table.public.id
}

# ── Private route tables (one per AZ → own NAT Gateway) ──
resource "aws_route_table" "private" {
  for_each = toset(var.azs)

  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main[each.key].id
  }

  tags = {
    Name      = "${var.project}-rt-private-${each.key}"
    Project   = var.project
    ManagedBy = "terraform"
  }
}

# ── Associate private subnets → their AZ's route table ───
resource "aws_route_table_association" "private" {
  for_each = aws_subnet.private

  subnet_id = each.value.id
  route_table_id = aws_route_table.private[
    var.azs[index(keys(aws_subnet.private), each.key)]
  ].id
}





