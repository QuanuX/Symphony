#include "pdf.hpp"
#include <iostream>
#include <stdexcept>
using symphony::knowledge::shv_pdf::parse_table;
int main() {
  const std::string heading =
      "69290 Rev. 1.00 April 2026\nTable 8. EPYC 7003 Series Portfolio APP "
      "Values\nOPN Model\nNumber\nAPP \n(Weighted TFLOPS)\n";
  std::string rows;
  for (int i = 0; i < 28; ++i)
    rows += "100-000000" + std::to_string(300 + i) + " " +
            std::to_string(7000 + i) + " 0.1000\n";
  auto good = heading + rows + "[Public]";
  if (parse_table(good).size() != 28)
    return 1;
  auto rejects = [](const std::string &s) {
    try {
      parse_table(s);
      return false;
    } catch (const std::exception &) {
      return true;
    }
  };
  if (!rejects(heading + rows.substr(rows.find('\n') + 1) + "[Public]"))
    return 2;
  if (!rejects(heading + rows + rows.substr(0, rows.find('\n') + 1) +
               "[Public]"))
    return 3;
  auto duplicate = good;
  duplicate.replace(duplicate.find("100-000000301"), 13, "100-000000300");
  if (!rejects(duplicate))
    return 4;
  if (!rejects(good + "unexpected"))
    return 5;
  if (!rejects(heading + heading + rows + "[Public]"))
    return 6;
  auto ambiguous = good;
  ambiguous.replace(ambiguous.find("0.1000"), 6, "unknown");
  if (!rejects(ambiguous))
    return 7;
  auto wrong = good;
  wrong.replace(wrong.find("Rev. 1.00"), 9, "Rev. 2.00");
  if (!rejects(wrong))
    return 8;
  // Reordering complete rows is preserved, never sorted into invented source
  // order.
  auto first = rows.substr(0, rows.find('\n') + 1);
  auto reordered = heading + rows.substr(first.size()) + first + "[Public]";
  if (parse_table(reordered).at(0).at("model") != "7001")
    return 9;
  std::cout << "bounded table parser: success and 7 rejection cases, source "
               "order preserved\n";
}
