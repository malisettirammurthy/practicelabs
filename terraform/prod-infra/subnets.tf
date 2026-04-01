# subnets.tf

# ── locals: build maps for for_each ─────────────────────
locals {
  public_subnets = zipmap(
    [for i, az in var.azs : "public-${index(var.azs, az) + 1}${substr(az, -1, 1)}"],
    [for i, cidr in var.public_subnet_cidrs : {
      cidr = cidr
      az   = var.azs[i]
    }]
  )

  private_subnets = zipmap(
    [for i, az in var.azs : "private-${index(var.azs, az) + 1}${substr(az, -1, 1)}"],
    [for i, cidr in var.private_subnet_cidrs : {
      cidr = cidr
      az   = var.azs[i]
    }]
  )

  isolated_subnets = zipmap(
    [for i, az in var.azs : "isolated-${index(var.azs, az) + 1}${substr(az, -1, 1)}"],
    [for i, cidr in var.isolated_subnet_cidrs : {
      cidr = cidr
      az   = var.azs[i]
    }]
  )
}

# ── public subnets ───────────────────────────────────────
resource "aws_subnet" "public" {
  for_each = local.public_subnets

  vpc_id                  = aws_vpc.main.id
  cidr_block              = each.value.cidr
  availability_zone       = each.value.az
  map_public_ip_on_launch = true

  tags = {
    Name    = "${var.project}-${each.key}"
    Tier    = "public"
    Project = var.project
    ManagedBy = "terraform"
  }
}

# ── private subnets ──────────────────────────────────────
resource "aws_subnet" "private" {
  for_each = local.private_subnets

  vpc_id                  = aws_vpc.main.id
  cidr_block              = each.value.cidr
  availability_zone       = each.value.az
  map_public_ip_on_launch = false

  tags = {
    Name    = "${var.project}-${each.key}"
    Tier    = "private"
    Project = var.project
    ManagedBy = "terraform"
  }
}

# ── isolated subnets ─────────────────────────────────────
resource "aws_subnet" "isolated" {
  for_each = local.isolated_subnets

  vpc_id                  = aws_vpc.main.id
  cidr_block              = each.value.cidr
  availability_zone       = each.value.az
  map_public_ip_on_launch = false

  tags = {
    Name    = "${var.project}-${each.key}"
    Tier    = "isolated"
    Project = var.project
    ManagedBy = "terraform"
  }
}