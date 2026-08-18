function refreshkeys()
{
   key1.gotoAndPlay(1);
   key2.gotoAndPlay(1);
   key3.gotoAndPlay(1);
   key4.gotoAndPlay(1);
   key5.gotoAndPlay(1);
   key6.gotoAndPlay(1);
   key1.keytext.text = codeToChar(_root.savedata2.data.controlarray[number][0]);
   key2.keytext.text = codeToChar(_root.savedata2.data.controlarray[number][1]);
   key3.keytext.text = codeToChar(_root.savedata2.data.controlarray[number][2]);
   key4.keytext.text = codeToChar(_root.savedata2.data.controlarray[number][3]);
   key5.keytext.text = codeToChar(_root.savedata2.data.controlarray[number][4]);
   key6.keytext.text = codeToChar(_root.savedata2.data.controlarray[number][5]);
}
function codeToChar(input)
{
   returnvalue = " ";
   switch(input)
   {
      case 8:
         returnvalue = "BS";
         break;
      case 13:
         returnvalue = "Entr";
         break;
      case 16:
         returnvalue = "Shft";
         break;
      case 17:
         returnvalue = "Ctrl";
         break;
      case 20:
         returnvalue = "Cpsl";
         break;
      case 32:
         returnvalue = "SPC";
         break;
      case 33:
         returnvalue = "PgU";
         break;
      case 34:
         returnvalue = "PgD";
         break;
      case 35:
         returnvalue = "End";
         break;
      case 36:
         returnvalue = "Hom";
         break;
      case 37:
         returnvalue = "Left Arrow";
         break;
      case 38:
         returnvalue = "Up Arrow";
         break;
      case 39:
         returnvalue = "Right Arrow";
         break;
      case 40:
         returnvalue = "Down Arrow";
         break;
      case 45:
         returnvalue = "Ins";
         break;
      case 46:
         returnvalue = "Del";
         break;
      case 145:
         returnvalue = "Scrl";
         break;
      case 96:
         returnvalue = "np0";
         break;
      case 97:
         returnvalue = "np1";
         break;
      case 98:
         returnvalue = "np2";
         break;
      case 99:
         returnvalue = "np3";
         break;
      case 100:
         returnvalue = "np4";
         break;
      case 101:
         returnvalue = "np5";
         break;
      case 102:
         returnvalue = "np6";
         break;
      case 103:
         returnvalue = "np7";
         break;
      case 104:
         returnvalue = "np8";
         break;
      case 105:
         returnvalue = "np9";
         break;
      case 106:
         returnvalue = "np*";
         break;
      case 107:
         returnvalue = "np+";
         break;
      case 109:
         returnvalue = "np-";
         break;
      case 110:
         returnvalue = "np.";
         break;
      case 111:
         returnvalue = "np/";
         break;
      case 113:
         returnvalue = "F2";
         break;
      case 115:
         returnvalue = "F4";
         break;
      case 118:
         returnvalue = "F7";
         break;
      case 119:
         returnvalue = "F8";
         break;
      case 120:
         returnvalue = "F9";
         break;
      case 121:
         returnvalue = "F10";
         break;
      case 123:
         returnvalue = "F12";
         break;
      case 186:
         returnvalue = ";";
         break;
      case 187:
         returnvalue = "=";
         break;
      case 188:
         returnvalue = ",";
         break;
      case 189:
         returnvalue = "-";
         break;
      case 190:
         returnvalue = ".";
         break;
      case 191:
         returnvalue = "/";
         break;
      case 192:
         returnvalue = "`";
         break;
      case 219:
         returnvalue = "[";
         break;
      case 220:
         returnvalue = "\\";
         break;
      case 221:
         returnvalue = "]";
         break;
      case 222:
         returnvalue = "\'";
   }
   if(input >= 65 && input <= 90 || input >= 48 && input <= 57)
   {
      returnvalue = String.fromCharCode(input);
   }
   return returnvalue;
}
stop();
switch(_name)
{
   case "p1":
      number = 0;
      break;
   case "p2":
      number = 1;
      break;
   case "p3":
      number = 2;
      break;
   case "p4":
      number = 3;
}
pnumber.gotoAndStop(number + 1);
refreshkeys();
